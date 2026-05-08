-- =============================================================================
-- init.lua — Full LazyVim-based Neovim configuration
-- =============================================================================
-- Bootstrap lazy.nvim plugin manager
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    "git",
    "clone",
    "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable",
    lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)

-- Basic editor options
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.tabstop = 2
vim.opt.shiftwidth = 2
vim.opt.expandtab = true
vim.opt.smartindent = true
vim.opt.wrap = false
vim.opt.termguicolors = true
vim.opt.mouse = "a"
vim.opt.clipboard = "unnamedplus"
vim.opt.ignorecase = true
vim.opt.smartcase = true
vim.opt.cursorline = true
vim.opt.signcolumn = "yes"
vim.opt.updatetime = 250
vim.opt.timeoutlen = 300
vim.opt.splitright = true
vim.opt.splitbelow = true
vim.opt.scrolloff = 8
vim.opt.sidescrolloff = 8
vim.opt.undofile = true
vim.opt.list = true
vim.opt.listchars = { tab = "» ", trail = "·", nbsp = "␣" }

-- Leader key
vim.g.mapleader = " "
vim.g.maplocalleader = " "

-- Basic keymaps
local km = vim.keymap.set
local opts = { noremap = true, silent = true }

km("n", "<leader>w", "<cmd>write<CR>", { desc = "Save file" })
km("n", "<leader>q", "<cmd>q<CR>", { desc = "Quit" })
km("n", "<leader>Q", "<cmd>qa<CR>", { desc = "Quit all" })
km("n", "<leader>e", "<cmd>Neotree toggle<CR>", { desc = "Toggle file tree" })
km("n", "<C-h>", "<C-w>h", { desc = "Window left" })
km("n", "<C-j>", "<C-w>j", { desc = "Window down" })
km("n", "<C-k>", "<C-w>k", { desc = "Window up" })
km("n", "<C-l>", "<C-w>l", { desc = "Window right" })
km("n", "<C-Up>", "<cmd>resize +2<CR>", { desc = "Resize up" })
km("n", "<C-Down>", "<cmd>resize -2<CR>", { desc = "Resize down" })
km("n", "<C-Left>", "<cmd>vertical resize -2<CR>", { desc = "Resize left" })
km("n", "<C-Right>", "<cmd>vertical resize +2<CR>", { desc = "Resize right" })
km("n", "<leader>h", "<cmd>nohlsearch<CR>", { desc = "Clear search highlights" })
km("n", "<leader>bb", "<cmd>BufferLinePick<CR>", { desc = "Pick buffer" })
km("n", "<leader>bd", "<cmd>bd<CR>", { desc = "Delete buffer" })
km("n", "<leader>bn", "<cmd>bnext<CR>", { desc = "Next buffer" })
km("n", "<leader>bp", "<cmd>bprevious<CR>", { desc = "Previous buffer" })

-- Plugin specifications
require("lazy").setup({
  -- Colorscheme
  {
    "folke/tokyonight.nvim",
    lazy = false,
    priority = 1000,
    config = function()
      require("tokyonight").setup({
        style = "storm",
        transparent = false,
        terminal_colors = true,
        styles = {
          comments = { italic = true },
          functions = { bold = true },
          keywords = { italic = true },
        },
      })
      vim.cmd.colorscheme("tokyonight")
    end,
  },

  -- LazyVim starter (core plugin set)
  { "LazyVim/LazyVim", import = "lazyvim.plugins" },

  -- Which-key: shows keybindings popup
  {
    "folke/which-key.nvim",
    event = "VeryLazy",
    config = function()
      require("which-key").setup({})
    end,
  },

  -- File tree
  {
    "nvim-neo-tree/neo-tree.nvim",
    branch = "v3.x",
    dependencies = {
      "nvim-lua/plenary.nvim",
      "nvim-tree/nvim-web-devicons",
      "MunifTanjim/nui.nvim",
    },
    cmd = "Neotree",
    config = function()
      require("neo-tree").setup({
        close_if_last_window = true,
        enable_git_status = true,
        window = {
          position = "left",
          width = 30,
        },
      })
    end,
  },

  -- Fuzzy finder (Telescope)
  {
    "nvim-telescope/telescope.nvim",
    branch = "0.1.x",
    dependencies = {
      "nvim-lua/plenary.nvim",
      { "nvim-telescope/telescope-fzf-native.nvim", build = "make" },
    },
    config = function()
      local telescope = require("telescope")
      telescope.setup({
        defaults = {
          file_ignore_patterns = { "node_modules", ".git" },
        },
        extensions = {
          fzf = {
            fuzzy = true,
            override_generic_solver = true,
            override_file_solver = true,
            case_mode = "smart_case",
          },
        },
      })
      telescope.load_extension("fzf")

      -- Telescope keymaps
      km("n", "<leader>ff", "<cmd>Telescope find_files<CR>", { desc = "Find files" })
      km("n", "<leader>fg", "<cmd>Telescope live_grep<CR>", { desc = "Live grep" })
      km("n", "<leader>fb", "<cmd>Telescope buffers<CR>", { desc = "Find buffers" })
      km("n", "<leader>fh", "<cmd>Telescope help_tags<CR>", { desc = "Help tags" })
      km("n", "<leader>fk", "<cmd>Telescope keymaps<CR>", { desc = "Find keymaps" })
      km("n", "<leader>fo", "<cmd>Telescope oldfiles<CR>", { desc = "Recent files" })
      km("n", "<leader>fs", "<cmd>Telescope lsp_document_symbols<CR>", { desc = "Document symbols" })
    end,
  },

  -- Treesitter (syntax highlighting)
  {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    config = function()
      require("nvim-treesitter.configs").setup({
        ensure_installed = { "go", "python", "lua", "vim", "vimdoc", "query", "yaml", "toml", "json", "markdown", "bash" },
        auto_install = true,
        highlight = { enable = true },
        indent = { enable = true },
      })
    end,
  },

  -- LSP support (mason + mason-lspconfig + nvim-lspconfig)
  {
    "williamboman/mason.nvim",
    build = ":MasonUpdate",
    config = function()
      require("mason").setup({})
    end,
  },
  {
    "williamboman/mason-lspconfig.nvim",
    dependencies = { "williamboman/mason.nvim" },
    config = function()
      require("mason-lspconfig").setup({
        ensure_installed = { "gopls", "pyright", "lua_ls", "tsserver" },
        automatic_installation = true,
      })
    end,
  },
  {
    "neovim/nvim-lspconfig",
    dependencies = {
      "williamboman/mason.nvim",
      "williamboman/mason-lspconfig.nvim",
    },
    config = function()
      local lspconfig = require("lspconfig")
      local capabilities = require("cmp_nvim_lsp").default_capabilities()

      -- LSP keymaps
      local on_attach = function(client, bufnr)
        local bufopts = { buffer = bufnr, noremap = true, silent = true }
        km("n", "gD", vim.lsp.buf.declaration, bufopts)
        km("n", "gd", vim.lsp.buf.definition, bufopts)
        km("n", "K", vim.lsp.buf.hover, bufopts)
        km("n", "gi", vim.lsp.buf.implementation, bufopts)
        km("n", "<C-k>", vim.lsp.buf.signature_help, bufopts)
        km("n", "<leader>wa", vim.lsp.buf.add_workspace_folder, bufopts)
        km("n", "<leader>wr", vim.lsp.buf.remove_workspace_folder, bufopts)
        km("n", "<leader>wl", function() print(vim.inspect(vim.lsp.buf.list_workspace_folders())) end, bufopts)
        km("n", "<leader>rn", vim.lsp.buf.rename, bufopts)
        km("n", "<leader>ca", vim.lsp.buf.code_action, bufopts)
        km("n", "gr", vim.lsp.buf.references, bufopts)
        km("n", "<leader>f", function() vim.lsp.buf.format({ async = true }) end, bufopts)
      end

      -- Configure LSP servers
      local servers = { "gopls", "pyright", "lua_ls", "tsserver" }
      for _, lsp in ipairs(servers) do
        lspconfig[lsp].setup({
          capabilities = capabilities,
          on_attach = on_attach,
        })
      end

      -- Gopls specific settings
      lspconfig.gopls.setup({
        capabilities = capabilities,
        on_attach = on_attach,
        settings = {
          gopls = {
            analyses = { unusedparams = true },
            staticcheck = true,
          },
        },
      })

      -- Pyright specific settings
      lspconfig.pyright.setup({
        capabilities = capabilities,
        on_attach = on_attach,
        settings = {
          pyright = {
            disableOrganizeImports = false,
          },
          python = {
            analysis = {
              typeCheckingMode = "basic",
            },
          },
        },
      })
    end,
  },

  -- Autocomplete (nvim-cmp)
  {
    "hrsh7th/nvim-cmp",
    event = "InsertEnter",
    dependencies = {
      "hrsh7th/cmp-nvim-lsp",
      "hrsh7th/cmp-buffer",
      "hrsh7th/cmp-path",
      "hrsh7th/cmp-cmdline",
      "L3MON4D3/LuaSnip",
      "saadparwaiz1/cmp_luasnip",
    },
    config = function()
      local cmp = require("cmp")
      local luasnip = require("luasnip")

      cmp.setup({
        snippet = {
          expand = function(args)
            luasnip.lsp_expand(args.body)
          end,
        },
        mapping = cmp.mapping.preset.insert({
          ["<C-b>"] = cmp.mapping.scroll_docs(-4),
          ["<C-f>"] = cmp.mapping.scroll_docs(4),
          ["<C-Space>"] = cmp.mapping.complete(),
          ["<C-e>"] = cmp.mapping.abort(),
          ["<CR>"] = cmp.mapping.confirm({ select = true }),
          ["<Tab>"] = cmp.mapping(function(fallback)
            if cmp.visible() then
              cmp.select_next_item()
            elseif luasnip.expand_or_jumpable() then
              luasnip.expand_or_jump()
            else
              fallback()
            end
          end, { "i", "s" }),
          ["<S-Tab>"] = cmp.mapping(function(fallback)
            if cmp.visible() then
              cmp.select_prev_item()
            elseif luasnip.jumpable(-1) then
              luasnip.jump(-1)
            else
              fallback()
            end
          end, { "i", "s" }),
        }),
        sources = cmp.config.sources({
          { name = "nvim_lsp" },
          { name = "luasnip" },
        }, {
          { name = "buffer" },
          { name = "path" },
        }),
      })

      -- / cmdline completion
      cmp.setup.cmdline("/", {
        mapping = cmp.mapping.preset.cmdline(),
        sources = { { name = "buffer" } },
      })

      -- : cmdline completion
      cmp.setup.cmdline(":", {
        mapping = cmp.mapping.preset.cmdline(),
        sources = cmp.config.sources({
          { name = "path" },
        }, {
          { name = "cmdline" },
        }),
      })
    end,
  },

  -- Buffer line (tab bar)
  {
    "akinsho/bufferline.nvim",
    dependencies = { "nvim-tree/nvim-web-devicons" },
    event = "VeryLazy",
    config = function()
      require("bufferline").setup({
        options = {
          mode = "buffers",
          separator_style = "thin",
          show_buffer_close_icons = true,
          show_close_icon = false,
        },
      })
    end,
  },

  -- Status line (lualine)
  {
    "nvim-lualine/lualine.nvim",
    dependencies = { "nvim-tree/nvim-web-devicons" },
    event = "VeryLazy",
    config = function()
      require("lualine").setup({
        options = {
          theme = "tokyonight",
          component_separators = { left = "", right = "" },
          section_separators = { left = "", right = "" },
        },
      })
    end,
  },

  -- Git signs in gutter
  {
    "lewis6991/gitsigns.nvim",
    event = "VeryLazy",
    config = function()
      require("gitsigns").setup({})
    end,
  },

  -- Vim-fugitive (Git commands inside Neovim)
  { "tpope/vim-fugitive", cmd = { "Git", "G" } },

  -- Comment toggling
  {
    "numToStr/Comment.nvim",
    keys = {
      { "gc", mode = { "n", "v" }, desc = "Comment toggle linewise" },
      { "gb", mode = { "n", "v" }, desc = "Comment toggle blockwise" },
    },
    config = function()
      require("Comment").setup({})
    end,
  },

  -- Indentation guides
  {
    "lukas-reineke/indent-blankline.nvim",
    event = "VeryLazy",
    config = function()
      require("ibl").setup({
        indent = { char = "│" },
        scope = { enabled = false },
      })
    end,
  },

  -- Auto pairs
  {
    "windwp/nvim-autopairs",
    event = "InsertEnter",
    config = function()
      require("nvim-autopairs").setup({})
    end,
  },
}, {
  install = { colorscheme = { "tokyonight" } },
  checker = { enabled = false },
  change_detection = { notify = false },
})
