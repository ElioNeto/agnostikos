#!/usr/bin/env node
/**
 * Stage all files for issue #24 (dotfiles) and commit.
 */
import { execSync } from 'child_process';

const files = [
  'agnostic.yaml.example',
  'internal/config/config.go',
  'internal/bootstrap/rootfs.go',
  'internal/bootstrap/rootfs_test.go',
  'cmd/agnostic/dotfiles.go',
  'internal/dotfiles/manager.go',
  'internal/dotfiles/manager_test.go',
  'configs/git/.gitconfig',
  'configs/git/.gitignore_global',
  'configs/neovim/init.lua',
  'configs/starship/starship.toml',
  'configs/alacritty/alacritty.toml',
  'configs/tmux/.tmux.conf',
  '.opencode/state/task-state.json',
];

try {
  execSync(`git add ${files.join(' ')}`, { stdio: 'inherit', cwd: process.cwd() });
  console.log('✅ Files staged successfully');
} catch (err) {
  console.error('❌ Failed to stage files:', err.message);
  process.exit(1);
}
