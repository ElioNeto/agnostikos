import { describe, it } from '@jest/globals';

describe('go.mod', () => {
  describe('happy path', () => {
    it('should import spf13/cobra v1.8.1', () => {
      expect(require('github.com/spf13/cobra').version).toBe('v1.8.1');
    });

    it('should import gopkg.in/yaml.v3 v3.0.1', () => {
      expect(require('gopkg.in/yaml.v3').Version).toBe('v3.0.1');
    });
  });

  describe('edge cases', () => {
    it('should handle invalid module path', () => {
      try {
        require('nonexistent-module-path');
      } catch (e) {
        expect(e.code).toBe('MODULE_NOT_FOUND');
      }
    });

    it('should handle version mismatch', () => {
      const cobraVersion = require('github.com/spf13/cobra').version;
      expect(cobraVersion.startsWith('v')).toBe(true);
      expect(cobraVersion.includes('.')).toBe(true);
      expect(cobraVersion.endsWith('.1')).toBe(true);
    });
  });
});