# Perfis do AgnosticOS

Cada perfil define um conjunto curado de pacotes para um propósito específico.

## Perfis disponíveis

| Perfil    | Descrição                                           |
|-----------|-----------------------------------------------------|
| `minimal` | Sistema base mínimo — apenas o essencial para rodar |
| `desktop` | Ambiente gráfico Wayland com Hyprland e ferramentas |
| `server`  | Servidor headless com foco em estabilidade          |
| `dev`     | Estação de desenvolvimento completa                 |

---

## Perfil `dev` — Categorias de ferramentas

### 1. Shell / Terminal
Ferramentas para produtividade no terminal.

| Pacote      | Descrição |
|-------------|-----------|
| `zsh`       | Shell poderoso com autocompletar, temas e plugins |
| `starship`  | Prompt minimalista, rápido e altamente customizável |
| `fzf`       | Fuzzy finder para arquivos, histórico, processos e muito mais |
| `tmux`      | Multiplexador de terminal — múltiplas sessões em uma única janela |
| `alacritty` | Terminal acelerado por GPU, configurado via TOML |

### 2. Version Control
Controle de versão e colaboração.

| Pacote    | Descrição |
|-----------|-----------|
| `git`     | Sistema de controle de versão distribuído |
| `lazygit` | Interface TUI para git com atalhos para diff, commit, branch |
| `gh`      | GitHub CLI — issues, pull requests, releases pelo terminal |
| `delta`   | Diff viewer com sintaxe colorida para git diff |

### 3. Runtimes / Language Toolchains
Gerenciamento de versões de linguagens de programação.

| Pacote   | Descrição |
|----------|-----------|
| `mise`   | Gerenciador de versões de runtime (Node, Python, Ruby, Java) |
| `rustup` | Instalador e versionador do Rust (rustc, cargo, rustfmt) |
| `go`     | Linguagem Go compilada com goroutines e binários estáticos |

### 4. Containers / Virtualização
Ferramentas para containers e ambientes isolados.

| Pacote          | Descrição |
|-----------------|-----------|
| `docker`        | Container runtime e orquestração padrão da indústria |
| `docker-compose` | Orquestração multi-container via YAML |
| `podman`        | Alternativa rootless ao Docker, sem daemon |
| `distrobox`     | Cria containers com distribuição completa integrados ao host |

### 5. Editores / IDEs
Editores de texto para desenvolvimento.

| Pacote    | Descrição |
|-----------|-----------|
| `neovim`  | Editor de texto extensível com Lua, LSP e Tree-sitter |
| `helix`   | Editor modal nativo com LSP embutido (escrito em Rust) |

### 6. Modern Utils (GNU replacements)
Substitutos modernos para utilitários clássicos do UNIX.

| Pacote     | Descrição |
|------------|-----------|
| `eza`      | `ls` moderno com ícones, cores e visualização em árvore |
| `bat`      | `cat` com syntax highlight, line numbers e git gutter |
| `ripgrep`  | `grep` recursivo extremamente rápido com regex |
| `fd`       | `find` simples e rápido com sintaxe intuitiva |
| `zoxide`   | `cd` inteligente que aprende diretórios mais usados |
| `dust`     | `du` visual com gráfico de barras |
| `procs`    | `ps` moderno com tree view, cores e busca |
| `httpie`   | `curl` humano — JSON formatado, cores, sessões |
| `yazi`     | Gerenciador de arquivos TUI com Vim keybindings |

### 7. Network / Debug
Ferramentas de rede e depuração.

| Pacote     | Descrição |
|------------|-----------|
| `curl`     | Transferência de dados via URL (HTTP, FTP, etc.) |
| `wget`     | Download recursivo de arquivos |
| `nmap`     | Scanner de portas e descoberta de rede |
| `netcat`   | Ferramenta TCP/IP para debugging de conexões |
| `mkcert`   | Cria certificados SSL/TLS locais confiáveis |
| `wireshark` | Analisador de pacotes de rede com interface gráfica |

### 8. Observabilidade / System
Monitoramento e análise de desempenho do sistema.

| Pacote   | Descrição |
|----------|-----------|
| `btop`   | Monitor de sistema com CPU, RAM, disco e processos (TUI) |
| `strace` | Rastreia chamadas de sistema (syscalls) de processos |
| `lsof`   | Lista arquivos abertos por processo ou socket |
| `perf`   | Profiler de CPU do kernel Linux (perf_events) |

### 9. Segurança / Criptografia
Ferramentas de segurança e gerenciamento de chaves.

| Pacote    | Descrição |
|-----------|-----------|
| `gnupg`   | Implementação OpenPGP para assinatura, cifragem e chaves |
| `pass`    | Gerenciador de senhas via GPG (password-store) |
| `age`     | Criptografia moderna e simples (substituto do OpenPGP) |
| `openssh` | Conexão remota segura (cliente e servidor SSH) |
