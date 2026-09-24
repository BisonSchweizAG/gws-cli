package update

import "syscall"

// hideFile marks path as hidden. Windows cannot delete the running executable,
// so after an update the old binary (".gws.exe.old") is kept but hidden.
func hideFile(path string) error {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return syscall.SetFileAttributes(name, syscall.FILE_ATTRIBUTE_HIDDEN)
}
