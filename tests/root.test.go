import { expect } from 'vitest';
import * as cobra from 'cobra';

describe('agnostic/root', () => {
  let rootCmd: cobra.Command;
  beforeEach(() => {
    rootCmd = new cobra.Command('agnostic');
  });

  describe('Execute function', () => {
    it('should exit with an error if Execute fails', async () => {
      const mockError = new Error('Failed to execute command');
      jest.spyOn(rootCmd, 'Execute').mockRejectedValue(mockError);
      await expect(rootCmd.Execute()).rejects.toThrowError(mockError);
      expect(process.exit).toHaveBeenCalledWith(1);
    });

    it('should exit with success if Execute succeeds', async () => {
      jest.spyOn(rootCmd, 'Execute').mockResolvedValue();
      await rootCmd.Execute();
      expect(process.exit).not.toHaveBeenCalledWith(1);
    });
  });

  describe('init function', () => {
    it('should add isolated command to root command', () => {
      init(rootCmd);
      expect(rootCmd.HasCommand('isolated')).toBe(true);
    });

    it('should set help func for isolated command', () => {
      const mockHelpFunc = jest.fn();
      init(rootCmd, undefined, mockHelpFunc);
      expect(isolatedCmd.GetHelpFunc()).toEqual(mockHelpFunc);
    });

    it('should disable default cmd for root command', () => {
      init(rootCmd);
      expect(rootCmd.CompletionOptions.DisableDefaultCmd).toBe(true);
    });
  });
});