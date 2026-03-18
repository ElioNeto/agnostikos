import { buildCmd } from '../cmd/agnostic/build';
import * as yaml from 'js-yaml';

describe('build command', () => {
  beforeEach(() => {
    jest.spyOn(os, 'readFile').mockResolvedValueOnce(`
      name: Base OS
      version: 1.0
      arch: x86_64
      description: A simple Linux distribution.
      packages:
        - package1
        - package2
      build:
        kernel_version: 5.10
        output_iso: custom.iso
        uefi: true
    `);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('should create rootfs, compile kernel and generate ISO', async () => {
    await expect(buildCmd.RunE()).resolves.not.toThrow();

    expect(bootstrap.CreateRootFS).toHaveBeenCalledWith('/mnt/lfs');
    expect(bootstrap.BuildKernel).toHaveBeenCalledWith({
      Version: '5.10',
      SourcesDir: '/mnt/lfs/sources',
      OutputDir: '/mnt/lfs/boot',
      Defconfig: 'x86_64_defconfig',
    });
    expect(bootstrap.GenerateISO).toHaveBeenCalledWith({
      Name: 'Base OS',
      Version: '1.0',
      RootFS: '/mnt/lfs',
      Output: 'custom.iso',
      UEFI: true,
      BootLabel: 'Base OS 1.0',
    });
  });

  it('should allow overriding output ISO path', async () => {
    jest.spyOn(buildCmd.Flags(), 'StringVarP').mockReturnValueOnce('my-custom-iso.iso');

    await expect(buildCmd.RunE()).resolves.not.toThrow();

    expect(bootstrap.GenerateISO).toHaveBeenCalledWith({
      Name: 'Base OS',
      Version: '1.0',
      RootFS: '/mnt/lfs',
      Output: 'my-custom-iso.iso',
      UEFI: true,
      BootLabel: 'Base OS 1.0',
    });
  });

  it('should use default output ISO path if not specified', async () => {
    await expect(buildCmd.RunE()).resolves.not.toThrow();

    expect(bootstrap.GenerateISO).toHaveBeenCalledWith({
      Name: 'Base OS',
      Version: '1.0',
      RootFS: '/mnt/lfs',
      Output: 'base_os_1.0.iso',
      UEFI: true,
      BootLabel: 'Base OS 1.0',
    });
  });

  it('should handle missing recipe file', async () => {
    jest.spyOn(os, 'readFile').mockRejectedValueOnce(new Error('file not found'));

    await expect(buildCmd.RunE()).rejects.toThrow();
  });
});