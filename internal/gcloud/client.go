package gcloud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	workstations "cloud.google.com/go/workstations/apiv1"
	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

var pollInterval = 10 * time.Second

var (
	clientMu     sync.Mutex
	cachedClient *workstations.Client
)

// getClient returns a shared, reusable workstations.Client, creating it on
// first use. Subsequent calls reuse the same client instance, avoiding the
// overhead of re-authenticating and re-dialing for every operation.
func getClient(ctx context.Context, cfg *types.Config) (*workstations.Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	tokenSource, err := Login(ctx, cfg)
	if err != nil {
		return nil, err
	}

	c, err := workstations.NewClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	cachedClient = c
	return cachedClient, nil
}

// CloseClient closes the shared workstations.Client, if one was created.
// It should be called once, when the application is shutting down.
func CloseClient() error {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient == nil {
		return nil
	}

	err := cachedClient.Close()
	cachedClient = nil
	return err
}

func StartWorkstation(ctx context.Context, cfg *types.Config, boostConfig string) error {
	sshContext, c, _, ws, err := setup(ctx, cfg)
	if err != nil {
		return err
	}

	start := time.Now()
	timeout := cfg.StartTimeout()

	switch ws.GetState() {
	case workstationspb.Workstation_STATE_STOPPED:
		swr := &workstationspb.StartWorkstationRequest{Name: ws.GetName()}
		if boostConfig != "" {
			log.Logf("🚀 Starting with boost config: %s", boostConfig)
			swr.BoostConfig = boostConfig
		}
		op, err := c.StartWorkstation(ctx, swr)
		if err != nil {
			log.Logf("Error starting workstation: %v", err)
			return err
		}
		spinny := spinner.Start(fmt.Sprintf("Waiting for workstation %s to start...", sshContext.GCloud.Name))
		defer spinny.Stop() // reset the terminal in case of a panic

		_, err = op.Wait(context.Background()) // create a new ctx to not cancel the start command against google
		spinny.Stop()
		if err != nil {
			log.Logf("Error waiting for workstation to start: %v", err)
			return err
		}
		log.Logf("Workstation started in %s %q", time.Since(start).String(), sshContext.GCloud.Name)
	case workstationspb.Workstation_STATE_RUNNING:
		log.Logf("Workstation running %q", sshContext.GCloud.Name)
	case workstationspb.Workstation_STATE_STARTING:
		spinny := spinner.Start(fmt.Sprintf("Workstation %s is already starting ...", sshContext.GCloud.Name))
		defer spinny.Stop() // reset the terminal in case of a panic

		err = waitForWorkstationRunning(ctx, c, ws, timeout)
		spinny.Stop()

		if err != nil {
			log.Logf("Error waiting for workstation to start: %v", err)
			return err
		}

		if ws.GetState() == workstationspb.Workstation_STATE_RUNNING {
			log.Logf("Workstation started in %s %q", time.Since(start).String(), sshContext.GCloud.Name)
		} else {
			log.Logf("Workstation is in unexpected state: %s", ws.GetState())
		}
	default:
	}
	return nil
}

// waitForWorkstationRunning polls the workstation status until it's running or timeout occurs.
// Returns error if the workstation fails to reach in running state within the specified timeout.
func waitForWorkstationRunning(
	ctx context.Context,
	c *workstations.Client,
	ws *workstationspb.Workstation,
	timeout time.Duration,
) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for workstation %s to start", ws.GetName())
		case <-ticker.C:
			updatedWs, err := c.GetWorkstation(ctx, &workstationspb.GetWorkstationRequest{Name: ws.GetName()})
			if err != nil {
				return fmt.Errorf("failed to get workstation status: %w", err)
			}

			if updatedWs.GetState() == workstationspb.Workstation_STATE_RUNNING {
				*ws = workstationspb.Workstation{
					Name:  updatedWs.GetName(),
					State: updatedWs.GetState(),
				}
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func setup(
	ctx context.Context,
	cfg *types.Config,
) (*types.Context, *workstations.Client, *workstationspb.WorkstationConfig, *workstationspb.Workstation, error) {
	sshContext := cfg.CurrentContext()
	if sshContext == nil || sshContext.GCloud == nil {
		log.Log("No gcloud config found")
		return nil, nil, nil, nil, nil
	}
	// gcloud auth application-default login
	// Default credentials: ${HOME}/.config/gcloud/application_default_credentials.json
	c, err := getClient(ctx, cfg)
	if err != nil {
		log.Logf("Error creating workstations client: %v", err)
		return nil, nil, nil, nil, err
	}
	wsConfig := fmt.Sprintf("projects/%s/locations/%s/workstationClusters/%s/workstationConfigs/%s",
		sshContext.GCloud.Project,
		sshContext.GCloud.Region,
		sshContext.GCloud.Cluster,
		sshContext.GCloud.Config,
	)
	wsName := fmt.Sprintf("%s/workstations/%s",
		wsConfig,
		sshContext.GCloud.Name,
	)

	wsc, err := c.GetWorkstationConfig(ctx, &workstationspb.GetWorkstationConfigRequest{Name: wsConfig})
	if err != nil {
		log.Logf("Error getting workstation config: %v", err)
		return nil, nil, nil, nil, err
	}

	ws, err := c.GetWorkstation(ctx, &workstationspb.GetWorkstationRequest{Name: wsName})
	if err != nil {
		log.Logf("Error getting workstation: %v", err)
		return nil, nil, nil, nil, err
	}
	return sshContext, c, wsc, ws, err
}

type Project struct {
	ID   string
	Name string
}

func ListProjects(ctx context.Context, cfg *types.Config) ([]Project, error) {
	tokenSource, err := Login(ctx, cfg)
	if err != nil {
		return nil, err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	var projects []Project
	err = svc.Projects.List().Pages(ctx, func(response *cloudresourcemanager.ListProjectsResponse) error {
		for _, p := range response.Projects {
			if p.LifecycleState == "ACTIVE" {
				projects = append(projects, Project{ID: p.ProjectId, Name: p.Name})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return projects, nil
}

type Workstation struct {
	Project string
	Region  string
	Cluster string
	Config  string
	Name    string
}

func ListWorkstations(ctx context.Context, cfg *types.Config, project string) ([]Workstation, error) {
	c, err := getClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var workstationsList []Workstation

	// Try to list usable workstations in the project using wildcards.
	it := c.ListUsableWorkstations(ctx, &workstationspb.ListUsableWorkstationsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-/workstationClusters/-/workstationConfigs/-", project),
	})
	for {
		ws, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Logf("Warning: ListUsableWorkstations failed, trying nested list: %v", err)
			return listWorkstationsNested(ctx, c, project)
		}

		workstationsList = append(workstationsList, parseWorkstationName(ws.GetName()))
	}

	return workstationsList, nil
}

func listWorkstationsNested(ctx context.Context, c *workstations.Client, project string) ([]Workstation, error) {
	var workstationsList []Workstation

	// 1. List all clusters in the project across all locations
	clusterIt := c.ListWorkstationClusters(ctx, &workstationspb.ListWorkstationClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", project),
	})
	for {
		cluster, err := clusterIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list workstation clusters: %w", err)
		}

		// 2. List configs in this cluster
		configIt := c.ListWorkstationConfigs(ctx, &workstationspb.ListWorkstationConfigsRequest{
			Parent: cluster.GetName(),
		})
		for {
			config, err := configIt.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				continue
			}

			// 3. List workstations in this config
			wsIt := c.ListWorkstations(ctx, &workstationspb.ListWorkstationsRequest{
				Parent: config.GetName(),
			})
			for {
				ws, err := wsIt.Next()
				if errors.Is(err, iterator.Done) {
					break
				}
				if err != nil {
					continue
				}
				workstationsList = append(workstationsList, parseWorkstationName(ws.GetName()))
			}
		}
	}

	return workstationsList, nil
}

func parseWorkstationName(name string) Workstation {
	// projects/{project}/locations/{location}/workstationClusters/{cluster}/workstationConfigs/{config}/workstations/{workstation}
	parts := strings.Split(name, "/")
	if len(parts) >= 10 {
		return Workstation{
			Project: parts[1],
			Region:  parts[3],
			Cluster: parts[5],
			Config:  parts[7],
			Name:    parts[9],
		}
	}
	return Workstation{}
}

func StopAllWorkstations(ctx context.Context, cfg *types.Config) error {
	var errs []error
	currContext := cfg.CurrentContextName
	defer func() { _ = cfg.SwitchContext(currContext, true) }()
	for name := range cfg.Contexts {
		if err := cfg.SwitchContext(name, true); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := StopWorkstation(ctx, cfg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func StopWorkstation(ctx context.Context, cfg *types.Config) error {
	sshContext, c, _, ws, err := setup(ctx, cfg)
	if err != nil {
		return err
	}

	if ws.GetState() != workstationspb.Workstation_STATE_STOPPED {
		start := time.Now()
		op, err := c.StopWorkstation(ctx, &workstationspb.StopWorkstationRequest{Name: ws.GetName()})
		if err != nil {
			log.Logf("Error stopping workstation: %v", err)
			return err
		}
		spinny := spinner.Start(fmt.Sprintf("Waiting for workstation %s to stop...", sshContext.GCloud.Name))
		defer spinny.Stop() // reset the terminal in case of a panic

		_, err = op.Wait(ctx)
		if err != nil {
			log.Logf("Error waiting for workstation to stop: %v", err)
			return err
		}
		spinny.Stop()
		log.Logf("Workstation stopped in %s %q", time.Since(start).String(), sshContext.GCloud.Name)
	} else {
		log.Logf("Workstation already stopped %q", sshContext.GCloud.Name)
	}
	return nil
}

type WorkstationState struct {
	Context          string
	Project          string
	Config           string
	Name             string
	State            workstationspb.Workstation_State
	Uptime           *time.Duration
	RunningTimeout   *time.Duration
	IdleTimeout      *time.Duration
	ExpectedShutdown *time.Time
}

func GetWorkstationStates(ctx context.Context, cfg *types.Config, contextFilter string) ([]WorkstationState, error) {
	var states []WorkstationState
	curr := cfg.CurrentContextName
	defer func() { _ = cfg.UseContext(curr) }()
	for name, ctxCfg := range cfg.Contexts {
		if ctxCfg.GCloud == nil {
			continue
		}
		if contextFilter != "" && name != contextFilter {
			continue
		}
		_ = cfg.UseContext(name)
		_, _, wsc, ws, err := setup(ctx, cfg)
		if err != nil {
			states = append(states, WorkstationState{
				Context: name,
				Project: ctxCfg.GCloud.Project,
				Config:  ctxCfg.GCloud.Config,
				Name:    ctxCfg.GCloud.Name,
				State:   workstationspb.Workstation_STATE_UNSPECIFIED,
			})
			continue
		}
		var uptime *time.Duration
		if ws.GetState() == workstationspb.Workstation_STATE_RUNNING && ws.GetStartTime() != nil {
			u := time.Since(ws.GetStartTime().AsTime()).Round(time.Second)
			uptime = &u
		}
		var runningTimeout, idleTimeout *time.Duration
		if wsc.GetRunningTimeout() != nil {
			rt := wsc.GetRunningTimeout().AsDuration()
			runningTimeout = &rt
		}
		if wsc.GetIdleTimeout() != nil {
			it := wsc.GetIdleTimeout().AsDuration()
			idleTimeout = &it
		}
		var expectedShutdown *time.Time
		if ws.GetState() == workstationspb.Workstation_STATE_RUNNING && ws.GetStartTime() != nil && runningTimeout != nil {
			es := ws.GetStartTime().AsTime().Add(*runningTimeout)
			expectedShutdown = &es
		}
		states = append(states, WorkstationState{
			Context:          name,
			Project:          ctxCfg.GCloud.Project,
			Config:           ctxCfg.GCloud.Config,
			Name:             ctxCfg.GCloud.Name,
			State:            ws.GetState(),
			Uptime:           uptime,
			RunningTimeout:   runningTimeout,
			IdleTimeout:      idleTimeout,
			ExpectedShutdown: expectedShutdown,
		})
	}
	return states, nil
}
