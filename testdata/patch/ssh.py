  @classmethod
  def Current(cls):
    """Retrieve the current environment.

    Returns:
      Environment, the active and current environment on this machine.
    """
    if platforms.OperatingSystem.IsWindows():
      suite = Suite.PUTTY
      bin_path = _SdkHelperBin()
    else:
      suite = Suite.OPENSSH
      bin_path = None
    return Environment(suite, bin_path)
