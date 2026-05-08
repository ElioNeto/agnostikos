# Config Reference

This document describes every field in the `agnostic.yaml` configuration file.

---

## Top-level fields

### `version`

- **Type:** `string`
- **Required:** yes
- **Description:** Schema version of the configuration file.
- **Validation:** must be non-empty.
- **Example:**
  ```yaml
  version: "1"
  ```

---

### `locale`

- **Type:** `string`
- **Required:** yes
- **Description:** System locale. Format: `<language>_<REGION>.<encoding>`.
- **Validation:** must match `/^[a-z]{2}_[A-Z]{2}\.[a-zA-Z0-9._-]+$/`.
- **Example:**
  ```yaml
  locale: pt_BR.UTF-8
  ```

---

### `timezone`

- **Type:** `string`
- **Required:** no
- **Description:** System timezone. Format: `<Region>/<City>` or a short name like `UTC`.
- **Validation:** if set, must match `/^[A-Za-z_]+(\/[A-Za-z_]+)?$/`.
- **Example:**
  ```yaml
  timezone: America/Sao_Paulo
  ```

---

### `profile`

- **Type:** `string`
- **Required:** no
- **Description:** Installation profile. When set, profile-specific packages are included automatically.
- **Valid values:** `minimal`, `desktop`, `server`, `dev`
- **Example:**
  ```yaml
  profile: dev
  ```

---

## `packages`

### `packages.base`

- **Type:** `[]string`
- **Required:** yes
- **Description:** Base system packages — always installed. At minimum should include kernel, firmware, shell, and core utilities.
- **Validation:** no empty entries allowed.
- **Example:**
  ```yaml
  packages:
    base:
      - base
      - linux
      - linux-firmware
      - zsh
      - git
      - openssh
      - curl
      - starship
  ```

---

### `packages.dev`

- **Type:** `[]string`
- **Required:** no
- **Description:** Development tooling — editors, runtimes, containers, CLI utilities.
- **Validation:** no empty entries allowed.
- **Example:**
  ```yaml
  packages:
    dev:
      - neovim
      - lazygit
      - fzf
      - ripgrep
      - bat
      - eza
      - zoxide
      - docker
      - mise
  ```

---

### `packages.desktop`

- **Type:** `[]string`
- **Required:** no
- **Description:** Desktop environment packages — compositor, bar, launcher, terminal, notifications, audio, network, bluetooth.
- **Validation:** no empty entries allowed.
- **Example:**
  ```yaml
  packages:
    desktop:
      - hyprland
      - waybar
      - alacritty
      - rofi-wayland
      - dunst
  ```

---

### `packages.extra`

- **Type:** `[]string`
- **Required:** no
- **Description:** Extra packages installed when the `--extra` flag is passed.
- **Validation:** no empty entries allowed.
- **Example:**
  ```yaml
  packages:
    extra:
      - docker
      - neovim
      - ripgrep
  ```

---

## `backends`

### `backends.default`

- **Type:** `string`
- **Required:** yes
- **Description:** Default package backend for installing packages.
- **Valid values:** `pacman`, `nix`, `flatpak`
- **Example:**
  ```yaml
  backends:
    default: pacman
  ```

---

### `backends.fallback`

- **Type:** `string`
- **Required:** no
- **Description:** Alternative backend if the default one fails.
- **Valid values:** `pacman`, `nix`, `flatpak`
- **Example:**
  ```yaml
  backends:
    fallback: flatpak
  ```

---

## `user`

### `user.name`

- **Type:** `string`
- **Required:** no
- **Description:** Username to create / configure for the system.
- **Example:**
  ```yaml
  user:
    name: dev
  ```

---

### `user.shell`

- **Type:** `string`
- **Required:** no
- **Description:** Login shell for the user. Must be an absolute path.
- **Validation:** if set, must start with `/`.
- **Example:**
  ```yaml
  user:
    shell: /bin/zsh
  ```

---

### `user.groups`

- **Type:** `[]string`
- **Required:** no
- **Description:** Supplementary groups for the user (e.g. `wheel`, `docker`, `video`).
- **Validation:** no empty entries allowed.
- **Example:**
  ```yaml
  user:
    groups: [wheel, docker, libvirt]
  ```

---

## `dotfiles`

### `dotfiles.source`

- **Type:** `string`
- **Required:** no
- **Description:** Git URL or absolute local path to a dotfiles repository. When empty, the built-in dotfiles from the `configs/` directory are used.
- **Validation:** if set, must start with `http://`, `https://`, `git@`, or `/`.
- **Example:**
  ```yaml
  dotfiles:
    source: https://github.com/ElioNeto/agnostikos
  ```

---

### `dotfiles.apply`

- **Type:** `bool`
- **Required:** no
- **Description:** Whether to automatically apply dotfiles during bootstrap.
- **Default:** `false`
- **Example:**
  ```yaml
  dotfiles:
    apply: true
  ```

---

## Validation summary

| Field | Required | Type | Validation |
|-------|----------|------|------------|
| `version` | yes | `string` | non-empty |
| `locale` | yes | `string` | regex: `<lang>_<REGION>.<encoding>` |
| `timezone` | no | `string` | regex: `<Region>/<City>` or `UTC` |
| `profile` | no | `string` | one of: `minimal`, `desktop`, `server`, `dev` |
| `packages.base` | yes | `[]string` | no empty entries |
| `packages.dev` | no | `[]string` | no empty entries |
| `packages.desktop` | no | `[]string` | no empty entries |
| `packages.extra` | no | `[]string` | no empty entries |
| `backends.default` | yes | `string` | one of: `pacman`, `nix`, `flatpak` |
| `backends.fallback` | no | `string` | one of: `pacman`, `nix`, `flatpak` |
| `user.name` | no | `string` | — |
| `user.shell` | no | `string` | absolute path (starts with `/`) |
| `user.groups` | no | `[]string` | no empty entries |
| `dotfiles.source` | no | `string` | git URL or absolute path |
| `dotfiles.apply` | no | `bool` | — |
