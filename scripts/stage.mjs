#!/usr/bin/env node
/**
 * Stage all files for the current issue and commit them.
 * Run via: npm run test-stage
 */
import { execSync } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..');

const files = [
  '.opencode/state/task-state.json',
  'agnostic.yaml.example',
  'cmd/agnostic/dotfiles.go',
  'configs/git/.gitconfig',
  'configs/git/.gitignore_global',
  'configs/neovim/init.lua',
  'configs/starship/starship.toml',
  'configs/alacritty/alacritty.toml',
  'configs/tmux/.tmux.conf',
  'internal/bootstrap/rootfs.go',
  'internal/config/config.go',
  'internal/dotfiles/manager.go',
  'internal/dotfiles/manager_test.go',
];

const run = (cmd, opts = {}) =>
  execSync(cmd, { cwd: repoRoot, stdio: 'inherit', ...opts });

// Stage files
try {
  run(`git add ${files.join(' ')}`);
  console.log('\n✅ Files staged successfully');
} catch (err) {
  console.error('❌ Failed to stage files:', err.message);
  process.exit(1);
}

// Check if there are staged changes
try {
  run('git diff --cached --quiet', { stdio: 'pipe' });
  console.log('Nothing to commit (no staged changes).');
  process.exit(0);
} catch {
  // exit code != 0 means there ARE changes, proceed to commit
}

// Commit
try {
  run('git commit -m "feat(#24): implement dotfiles management system with CLI, bootstrap integration, and config stubs"');
  console.log('\n✅ Commit created successfully');
} catch (err) {
  console.error('❌ Failed to commit:', err.message);
  process.exit(1);
}
