package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/derekurban/forage/internal/apperr"
	"github.com/derekurban/forage/internal/capability"
	"github.com/derekurban/forage/internal/config"
	"github.com/derekurban/forage/internal/credentials"
	"github.com/derekurban/forage/internal/doctor"
	"github.com/derekurban/forage/internal/envfile"
	"github.com/derekurban/forage/internal/output"
	"github.com/derekurban/forage/internal/providers"
	"github.com/derekurban/forage/internal/quota"
	"github.com/derekurban/forage/internal/router"
	"github.com/derekurban/forage/internal/state"
	"github.com/derekurban/forage/internal/version"
)

type app struct {
	opts        output.Options
	commandPath string
}

func Execute() int {
	if err := envfile.Load(); err != nil {
		_ = output.WriteError(os.Stderr, output.Options{}, "forage", apperr.New(apperr.CodeInvalidConfig, "Failed to load .env: "+err.Error(), apperr.ExitInvalidArgsOrConfig))
		return apperr.ExitInvalidArgsOrConfig
	}
	a := &app{}
	root := a.rootCmd()
	if err := root.Execute(); err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			commandPath := a.commandPath
			if commandPath == "" {
				commandPath = root.CommandPath()
			}
			_ = output.WriteError(os.Stderr, a.opts, commandPath, ae)
			return ae.ExitCode
		}
		_ = output.WriteError(os.Stderr, a.opts, root.CommandPath(), apperr.New(apperr.CodeGeneral, err.Error(), apperr.ExitGeneral))
		return apperr.ExitGeneral
	}
	return 0
}

func (a *app) rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "forage",
		Short:         "Complementary research retrieval CLI for extraction, scholarly lookup, and archive checks",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			a.commandPath = cmd.CommandPath()
		},
	}
	cmd.PersistentFlags().BoolVar(&a.opts.JSON, "json", false, "emit JSON output")
	cmd.PersistentFlags().BoolVar(&a.opts.JSONL, "jsonl", false, "emit JSONL output")
	cmd.PersistentFlags().BoolVar(&a.opts.NoColor, "no-color", false, "disable styled terminal output")
	cmd.PersistentFlags().BoolVarP(&a.opts.Verbose, "verbose", "v", false, "include additional diagnostics")
	cmd.AddCommand(a.extractCmd(), a.scholarCmd(), a.archiveCmd(), a.configCmd(), a.setupCmd(), a.credentialsCmd(), a.providersCmd(), a.cacheCmd(), a.versionCmd())
	return cmd
}

func (a *app) configCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage forage config"}
	var overwrite bool
	init := &cobra.Command{
		Use:   "init",
		Short: "Create .forage/config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, created, err := config.Init(overwrite)
			if err != nil {
				return err
			}
			data := map[string]any{"path": path, "created": created}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Forage config"))
			if created {
				fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s already exists\n", path)
			}
			return nil
		},
	}
	init.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing config")
	cmd.AddCommand(init)
	cmd.AddCommand(&cobra.Command{
		Use:   "repair",
		Short: "Repair .forage/config.yaml defaults and removed providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, changed, err := config.Repair()
			if err != nil {
				if os.IsNotExist(err) {
					return apperr.MissingConfig(config.Path())
				}
				return err
			}
			data := map[string]any{"path": config.Path(), "changed": changed, "providers": len(cfg.Providers), "routing": len(cfg.Routing)}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Forage config"))
			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Repaired %s\n", config.Path())
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s already matches current defaults\n", config.Path())
			}
			return nil
		},
	})
	return cmd
}

func (a *app) setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Initialize config and store provider credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := config.Init(false); err != nil {
				return err
			}
			store := credentials.NewKeychainStore()
			apiProviders := []providers.Provider{}
			for _, p := range providers.Sorted() {
				if len(p.CredentialFields) > 0 && p.SetupGroup != "legacy_optional" {
					apiProviders = append(apiProviders, p)
				}
			}
			var saved []string
			currentGroup := ""
			for _, p := range apiProviders {
				if p.SetupGroup != currentGroup {
					currentGroup = p.SetupGroup
					fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Setup: "+strings.ReplaceAll(currentGroup, "_", " ")))
				}
				var add bool
				form := huh.NewForm(huh.NewGroup(
					huh.NewConfirm().
						Title("Configure " + p.Name + "?").
						Description(fmt.Sprintf("%s\nEnv fallback: %s\nSetup: %s", p.FreeTier, p.EnvVar, p.SetupURL)).
						Value(&add),
				))
				if err := form.Run(); err != nil {
					return err
				}
				if !add {
					continue
				}
				for _, f := range p.CredentialFields {
					var value string
					input := huh.NewInput().
						Title(f.Description).
						Value(&value)
					if f.Secret {
						input = input.EchoMode(huh.EchoModePassword)
					}
					form = huh.NewForm(huh.NewGroup(input))
					if err := form.Run(); err != nil {
						return err
					}
					value = strings.TrimSpace(value)
					if value == "" {
						if f.Required {
							return fmt.Errorf("%s.%s is required", p.ID, f.Name)
						}
						continue
					}
					if err := store.SetField(p.ID, f.Name, value); err != nil {
						return err
					}
				}
				saved = append(saved, p.ID)
			}
			data := map[string]any{"config": config.Path(), "credential_store": "keychain", "saved": saved}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Forage setup"))
			fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\nCredential store: OS keychain\nSaved credentials: %d\n", config.Path(), len(saved))
			return nil
		},
	}
}

func (a *app) providersCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "providers", Short: "Inspect provider registry, health, and quota"}
	cmd.AddCommand(a.providersListCmd(), a.providersDoctorCmd(), a.providersQuotaCmd())
	return cmd
}

func (a *app) providersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ps := providers.Sorted()
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), ps, nil)
			}
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Provider", "Auth", "Status", "Capabilities", "Free tier"})
			for _, p := range ps {
				t.AppendRow(table.Row{p.ID, p.AuthType, output.Status(a.opts, string(p.Status)), strings.Join(p.Capabilities, ", "), p.FreeTier})
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Provider registry"))
			t.Render()
			return nil
		},
	}
}

func (a *app) providersDoctorCmd() *cobra.Command {
	var all bool
	var capFilter string
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Check provider configuration and health",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, st, cleanup, err := a.loadConfiguredState()
			if err != nil {
				return err
			}
			defer cleanup()
			r := doctor.New(credentials.NewKeychainStore(), st)
			ctx, cancel := context.WithTimeout(cmd.Context(), 45*time.Second)
			defer cancel()
			var results []doctor.Result
			for _, p := range providers.Sorted() {
				if len(args) == 1 && p.ID != args[0] {
					continue
				}
				if capFilter != "" && !contains(p.Capabilities, capFilter) {
					continue
				}
				if p.Status == providers.MetadataOnly && !a.opts.Verbose && !all && len(args) == 0 && capFilter == "" {
					continue
				}
				results = append(results, r.Check(ctx, cfg, p))
			}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), results, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Provider doctor"))
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Provider", "Config", "Health", "Credential", "Message"})
			for _, res := range results {
				t.AppendRow(table.Row{res.Provider.ID, output.Status(a.opts, res.ConfigStatus), output.Status(a.opts, res.HealthStatus), res.CredentialSource, res.Message})
			}
			t.Render()
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "include metadata-only providers")
	c.Flags().StringVar(&capFilter, "capability", "", "check providers for a specific capability")
	return c
}

func (a *app) providersQuotaCmd() *cobra.Command {
	var provider string
	var reset string
	var preflight bool
	c := &cobra.Command{
		Use:   "quota",
		Short: "Show persisted provider quota and health state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, st, cleanup, err := a.loadConfiguredState()
			if err != nil {
				return err
			}
			defer cleanup()
			if reset != "" {
				if err := st.ResetProviderState(reset); err != nil {
					return err
				}
				return a.writeData(cmd, map[string]any{"reset": reset}, nil)
			}
			if preflight {
				svc := quota.New(cfg, st, credentials.NewKeychainStore())
				var results []quota.Result
				if provider != "" {
					results = append(results, svc.Preflight(cmd.Context(), provider))
				} else {
					for _, p := range providers.Sorted() {
						if p.Quota.CanPreflight {
							results = append(results, svc.Preflight(cmd.Context(), p.ID))
						}
					}
				}
				if a.opts.JSON || a.opts.JSONL {
					return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), results, nil)
				}
				fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Provider quota preflight"))
				t := table.NewWriter()
				t.SetOutputMirror(cmd.OutOrStdout())
				t.AppendHeader(table.Row{"Provider", "Status", "Message", "Remaining"})
				for _, res := range results {
					remaining := ""
					if res.State.Remaining != nil {
						remaining = fmt.Sprintf("%d", *res.State.Remaining)
						if res.State.Limit != nil {
							remaining = fmt.Sprintf("%d/%d", *res.State.Remaining, *res.State.Limit)
						}
					}
					t.AppendRow(table.Row{res.Provider, output.Status(a.opts, res.Status), res.Message, remaining})
				}
				t.Render()
				return nil
			}
			var states []state.ProviderState
			if provider != "" {
				ps, ok, err := st.ProviderState(provider)
				if err != nil {
					return err
				}
				if ok {
					states = []state.ProviderState{ps}
				} else {
					states = []state.ProviderState{}
				}
			} else {
				var err error
				states, err = st.ProviderStates()
				if err != nil {
					return err
				}
				filtered := states[:0]
				for _, ps := range states {
					if _, ok := providers.ByID(ps.Provider); ok {
						filtered = append(filtered, ps)
					}
				}
				states = filtered
			}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), states, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Provider quota state"))
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Provider", "Status", "Reason", "Remaining", "Retry/Reset", "Checked"})
			for _, ps := range states {
				remaining := ""
				if ps.Remaining != nil {
					remaining = fmt.Sprintf("%d", *ps.Remaining)
					if ps.Limit != nil {
						remaining = fmt.Sprintf("%d/%d", *ps.Remaining, *ps.Limit)
					}
					if ps.Used != nil {
						remaining += fmt.Sprintf(" used:%d", *ps.Used)
					}
				}
				retry := ps.RetryAfter
				if retry == "" {
					retry = ps.ResetAt
				}
				t.AppendRow(table.Row{ps.Provider, output.Status(a.opts, ps.Status), ps.Reason, remaining, retry, ps.LastCheckedAt})
			}
			t.Render()
			return nil
		},
	}
	c.Flags().StringVar(&provider, "provider", "", "show one provider")
	c.Flags().StringVar(&reset, "reset-local", "", "clear local quota state for provider")
	c.Flags().BoolVar(&preflight, "preflight", false, "refresh quota from provider preflight endpoints where available")
	return c
}

func (a *app) cacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Inspect local cache"}
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show cache status",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, st, cleanup, err := a.loadConfiguredState()
			if err != nil {
				return err
			}
			defer cleanup()
			stats, err := st.Stats()
			if err != nil {
				return err
			}
			stats["database"] = config.DBPath()
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), stats, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, "Cache status"))
			fmt.Fprintf(cmd.OutOrStdout(), "Database: %s\nProvider states: %v\nRecords: %v\n", stats["database"], stats["provider_state_count"], stats["record_count"])
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Clear cached records and raw payloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, st, cleanup, err := a.loadConfiguredState()
			if err != nil {
				return err
			}
			defer cleanup()
			if err := st.ClearCache(); err != nil {
				return err
			}
			return a.writeData(cmd, map[string]any{"cleared": true}, nil)
		},
	})
	return cmd
}

func (a *app) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print forage version",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := map[string]string{"version": version.Version, "commit": version.Commit, "date": version.Date, "go": "go1.26"}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "forage %s (%s, %s)\n", version.Version, version.Commit, version.Date)
			return nil
		},
	}
}

func (a *app) writeSearchTable(cmd *cobra.Command, resp capability.SearchResponse) error {
	fmt.Fprintln(cmd.OutOrStdout(), output.Heading(a.opts, resp.Capability))
	t := table.NewWriter()
	t.SetOutputMirror(cmd.OutOrStdout())
	t.AppendHeader(table.Row{"#", "Title", "Provider", "Domain", "URL"})
	for i, r := range resp.Results {
		t.AppendRow(table.Row{i + 1, r.Title, r.Provider, r.SourceDomain, r.URL})
	}
	t.Render()
	if resp.Routing != nil && a.opts.Verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Providers used: %s\n", strings.Join(resp.Routing.ProvidersUsed, ", "))
	}
	return nil
}

func (a *app) writeData(cmd *cobra.Command, data any, diagnostics any) error {
	if a.opts.JSON || a.opts.JSONL {
		return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, diagnostics)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func (a *app) extractCmd() *cobra.Command {
	var stdin bool
	var cacheMode string
	var explain bool
	cmd := &cobra.Command{
		Use:   "extract [URL_OR_FILE]",
		Short: "Extract URLs into readable documents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var urls []string
			if stdin {
				b, _ := io.ReadAll(cmd.InOrStdin())
				urls = lines(string(b))
			} else if len(args) == 1 && strings.HasPrefix(args[0], "http") {
				urls = []string{args[0]}
			} else if len(args) == 1 {
				b, err := os.ReadFile(args[0])
				if err != nil {
					return err
				}
				urls = lines(string(b))
			}
			cfg, st, cleanup, err := a.loadConfiguredState()
			if err != nil {
				return err
			}
			defer cleanup()
			r := router.New(cfg, st, credentials.NewKeychainStore())
			var docs []capability.FetchResponse
			for _, u := range urls {
				resp, ae := r.Fetch(cmd.Context(), capability.FetchRequest{URL: u, CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose})
				if ae != nil && len(urls) == 1 {
					return ae
				}
				if ae == nil {
					docs = append(docs, resp)
				}
			}
			return a.writeData(cmd, docs, nil)
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read URLs from stdin")
	cmd.Flags().StringVar(&cacheMode, "cache", "", "cache mode: auto, refresh, only")
	cmd.Flags().BoolVar(&explain, "explain-routing", false, "include routing diagnostics")
	return cmd
}

func (a *app) credentialsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "credentials", Short: "Manage provider credentials"}
	store := credentials.NewKeychainStore()
	cmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List credential status",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rows []map[string]any
			for _, p := range providers.Sorted() {
				if len(p.CredentialFields) == 0 {
					continue
				}
				for _, f := range p.CredentialFields {
					cred, _ := store.GetField(p.ID, f.Name, f.EnvVar)
					rows = append(rows, map[string]any{"provider": p.ID, "field": f.Name, "auth_type": p.AuthType, "env_var": f.EnvVar, "configured": cred.Found, "source": cred.Source, "required": f.Required})
				}
			}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), rows, nil)
			}
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Provider", "Field", "Auth", "Required", "Configured", "Source", "Env"})
			for _, r := range rows {
				t.AppendRow(table.Row{r["provider"], r["field"], r["auth_type"], r["required"], output.Status(a.opts, fmt.Sprint(r["configured"])), r["source"], r["env_var"]})
			}
			t.Render()
			return nil
		},
	})
	var fromEnv string
	var valueStdin bool
	var field string
	set := &cobra.Command{
		Use: "set PROVIDER", Short: "Store a provider credential", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := providers.ByID(args[0])
			if !ok {
				return fmt.Errorf("unknown provider %s", args[0])
			}
			f, ok := resolveCredentialField(p, field)
			if !ok {
				return fmt.Errorf("unknown credential field %q for %s", field, p.ID)
			}
			var val string
			envName := fromEnv
			if envName == "" {
				envName = f.EnvVar
			}
			if envName != "" {
				val = os.Getenv(envName)
			}
			if valueStdin {
				b, _ := io.ReadAll(cmd.InOrStdin())
				val = strings.TrimSpace(string(b))
			}
			if val == "" {
				return fmt.Errorf("no credential value supplied; use --from-env or --value-stdin")
			}
			if err := store.SetField(p.ID, f.Name, val); err != nil {
				return err
			}
			data := map[string]any{"provider": p.ID, "field": f.Name, "stored": true}
			if a.opts.JSON || a.opts.JSONL {
				return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stored credential for %s.%s\n", p.ID, f.Name)
			return nil
		},
	}
	set.Flags().StringVar(&field, "field", "", "credential field name; defaults to provider primary field")
	set.Flags().StringVar(&fromEnv, "from-env", "", "read credential from this environment variable; defaults to the provider field env var")
	set.Flags().BoolVar(&valueStdin, "value-stdin", false, "read credential from stdin")
	cmd.AddCommand(set)
	cmd.AddCommand(&cobra.Command{Use: "check PROVIDER", Short: "Check credential presence", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		p, ok := providers.ByID(args[0])
		if !ok {
			return fmt.Errorf("unknown provider %s", args[0])
		}
		var rows []map[string]any
		for _, f := range p.CredentialFields {
			c, _ := store.GetField(p.ID, f.Name, f.EnvVar)
			rows = append(rows, map[string]any{"provider": p.ID, "field": f.Name, "configured": c.Found, "source": c.Source, "required": f.Required})
		}
		if a.opts.JSON || a.opts.JSONL {
			return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), rows, nil)
		}
		for _, r := range rows {
			fmt.Fprintf(cmd.OutOrStdout(), "%s.%s configured=%v source=%s\n", r["provider"], r["field"], r["configured"], r["source"])
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "remove PROVIDER", Short: "Remove provider credential", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		p, ok := providers.ByID(args[0])
		if !ok {
			return fmt.Errorf("unknown provider %s", args[0])
		}
		removed := []string{}
		for _, f := range p.CredentialFields {
			err := store.DeleteField(p.ID, f.Name)
			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
				return err
			}
			removed = append(removed, f.Name)
		}
		data := map[string]any{"provider": p.ID, "removed": removed}
		if a.opts.JSON || a.opts.JSONL {
			return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), data, nil)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed credentials for %s\n", p.ID)
		return nil
	}})
	return cmd
}

func (a *app) scholarCmd() *cobra.Command {
	var limit int
	var cacheMode string
	var explain bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "scholar QUERY",
		Short: "Search and enrich scholarly literature",
		Long:  "Search and enrich scholarly literature through scholarly APIs. This complements native web search with paper metadata, DOI lookup, open-access checks, and citation expansion.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSearch(cmd, capability.SearchScholar, args[0], nil, nil, limit, "", "", cacheMode, explain || a.opts.Verbose, raw)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum scholarly records")
	cmd.PersistentFlags().StringVar(&cacheMode, "cache", "", "cache mode: auto, refresh, only")
	cmd.PersistentFlags().BoolVar(&explain, "explain-routing", false, "include routing diagnostics")
	cmd.PersistentFlags().BoolVar(&raw, "raw", false, "include raw provider payloads in JSON output")
	cmd.AddCommand(&cobra.Command{
		Use:   "doi DOI",
		Short: "Enrich a DOI with scholarly metadata and open-access data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cleanup, err := a.router()
			if err != nil {
				return err
			}
			defer cleanup()
			resp, ae := r.EnrichDOI(cmd.Context(), capability.DataRequest{ID: args[0], Query: args[0], CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose, IncludeRaw: raw})
			if ae != nil {
				return ae
			}
			return a.writeData(cmd, resp, nil)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "paper ID_OR_DOI",
		Short: "Enrich a scholarly work by provider ID or DOI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cleanup, err := a.router()
			if err != nil {
				return err
			}
			defer cleanup()
			resp, ae := r.EnrichPaper(cmd.Context(), capability.DataRequest{ID: args[0], Query: args[0], CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose, IncludeRaw: raw})
			if ae != nil {
				return ae
			}
			return a.writeData(cmd, resp, nil)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "citations DOI",
		Short: "Look up citation metadata for a DOI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cleanup, err := a.router()
			if err != nil {
				return err
			}
			defer cleanup()
			resp, ae := r.Citations(cmd.Context(), capability.DataRequest{ID: args[0], Query: args[0], CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose, IncludeRaw: raw})
			if ae != nil {
				return ae
			}
			return a.writeData(cmd, resp, nil)
		},
	})
	return cmd
}

func (a *app) archiveCmd() *cobra.Command {
	var cacheMode string
	var explain bool
	var limit int
	cmd := &cobra.Command{
		Use:   "archive URL",
		Short: "Check historical availability for a URL",
		Long:  "Check whether a URL has historical records through Internet Archive and Common Crawl. This complements native web search with source verification and historical availability.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, cleanup, err := a.router()
			if err != nil {
				return err
			}
			defer cleanup()
			resp, ae := r.ArchiveLookup(cmd.Context(), capability.DataRequest{URL: args[0], Limit: limit, CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose})
			if ae != nil {
				return ae
			}
			return a.writeData(cmd, resp, nil)
		},
	}
	cmd.Flags().StringVar(&cacheMode, "cache", "", "cache mode: auto, refresh, only")
	cmd.Flags().BoolVar(&explain, "explain-routing", false, "include routing diagnostics")
	cmd.Flags().IntVar(&limit, "limit", 5, "maximum archive records")
	return cmd
}

func (a *app) runSearch(cmd *cobra.Command, cap, query string, include, exclude []string, limit int, freshness, site, cacheMode string, explain bool, includeRaw bool) error {
	cfg, st, cleanup, err := a.loadConfiguredState()
	if err != nil {
		return err
	}
	defer cleanup()
	r := router.New(cfg, st, credentials.NewKeychainStore())
	resp, ae := r.Search(cmd.Context(), capability.SearchRequest{Query: query, Capability: cap, Limit: limit, Freshness: freshness, Site: site, Providers: include, ExcludeProviders: exclude, CacheMode: cacheMode, ExplainRouting: explain || a.opts.Verbose, IncludeRaw: includeRaw})
	if ae != nil {
		return ae
	}
	if a.opts.JSON || a.opts.JSONL {
		return output.Write(cmd.OutOrStdout(), a.opts, cmd.CommandPath(), resp, nil)
	}
	return a.writeSearchTable(cmd, resp)
}

func preview(markdown, text string) string {
	s := markdown
	if s == "" {
		s = text
	}
	if len(s) > 1200 {
		return s[:1200] + "..."
	}
	return s
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func resolveCredentialField(p providers.Provider, name string) (providers.CredentialField, bool) {
	if len(p.CredentialFields) == 0 {
		return providers.CredentialField{}, false
	}
	if name == "" {
		for _, f := range p.CredentialFields {
			if f.Name == "api_key" || f.Required {
				return f, true
			}
		}
		return p.CredentialFields[0], true
	}
	for _, f := range p.CredentialFields {
		if f.Name == name {
			return f, true
		}
	}
	return providers.CredentialField{}, false
}

func (a *app) loadConfiguredState() (config.Config, *state.Store, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return config.Config{}, nil, func() {}, apperr.MissingConfig(config.Path())
		}
		return config.Config{}, nil, func() {}, apperr.New(apperr.CodeInvalidConfig, err.Error(), apperr.ExitInvalidArgsOrConfig)
	}
	st, err := state.Open(cfg.Cache.Database)
	if err != nil {
		return config.Config{}, nil, func() {}, err
	}
	return cfg, st, func() { _ = st.Close() }, nil
}

func (a *app) router() (router.Router, func(), error) {
	cfg, st, cleanup, err := a.loadConfiguredState()
	if err != nil {
		return router.Router{}, cleanup, err
	}
	return router.New(cfg, st, credentials.NewKeychainStore()), cleanup, nil
}
