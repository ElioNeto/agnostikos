```typescript
import { describe, it, expect } from 'jest';
import cobra from 'cobra';

describe('agnostic/install', () => {
  const rootCmd = new cobra.Command('root');

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('installCmd', () => {
    it('should print a message and install the package via the specified backend', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Install: jest.fn() } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('install');
      cmd.setParent(rootCmd);
      await cmd.runE(['package']);
      expect(managerMock).toHaveBeenCalledWith();
      expect(managerMock().Backends.pacman.Install).toHaveBeenCalledWith('package');
    });

    it('should print an error message if the backend is not found', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: {},
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('install');
      cmd.setParent(rootCmd);
      await expect(cmd.runE(['package'])).rejects.toThrowError(/backend 'pacman' not found — available: pacman, nix, flatpak/);
    });

    it('should run in isolated namespace when the --isolated flag is set', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Install: jest.fn() } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('install');
      cmd.setParent(rootCmd);
      await cmd.runE(['--isolated', 'package']);
      expect(managerMock()).toHaveBeenCalledWith();
      expect(managerMock().Backends.pacman.Install).toHaveBeenCalledWith('package');
      expect(console.log).toHaveBeenCalledWith('🔒 Running in isolated namespace...');
    });

    it('should handle installation errors and print an error message', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Install: jest.fn().mockRejectedValue(new Error('installation failed')) } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('install');
      cmd.setParent(rootCmd);
      await expect(cmd.runE(['package'])).rejects.toThrowError(/installation failed/);
    });
  });

  describe('removeCmd', () => {
    it('should print a message and remove the package via the specified backend', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Remove: jest.fn() } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('remove');
      cmd.setParent(rootCmd);
      await cmd.runE(['package']);
      expect(managerMock()).toHaveBeenCalledWith();
      expect(managerMock().Backends.pacman.Remove).toHaveBeenCalledWith('package');
    });

    it('should print an error message if the backend is not found', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: {},
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('remove');
      cmd.setParent(rootCmd);
      await expect(cmd.runE(['package'])).rejects.toThrowError(/backend 'pacman' not found/);
    });

    it('should handle removal errors and print an error message', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Remove: jest.fn().mockRejectedValue(new Error('removal failed')) } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('remove');
      cmd.setParent(rootCmd);
      await expect(cmd.runE(['package'])).rejects.toThrowError(/removal failed/);
    });
  });

  describe('updateCmd', () => {
    it('should print a message and update all packages via the specified backend', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: { pacman: { Update: jest.fn() } },
      }));
      global.manager.NewAgnosticManager = managerMock;
      const cmd = new cobra.Command('update');
      cmd.setParent(rootCmd);
      await cmd.runE([]);
      expect(managerMock()).toHaveBeenCalledWith();
      expect(managerMock().Backends.pacman.Update).toHaveBeenCalled();
    });

    it('should print an error message if the backend is not found', async () => {
      const managerMock = jest.fn().mockImplementationOnce(() => ({
        Backends: {},
      }));
      global.manager.NewA