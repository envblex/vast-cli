package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/takayoshi/vast-cli/pkg/api"
	"github.com/takayoshi/vast-cli/pkg/client"
	"github.com/takayoshi/vast-cli/pkg/config"
	"github.com/takayoshi/vast-cli/pkg/display"
)

type GlobalFlags struct {
	APIKey  string
	URL     string
	Raw     bool
	Explain bool
	Curl    bool
	Retry   int
	Version bool
	Help    bool
}

func printHelp() {
	helpText := fmt.Sprintf(`vastai-go %s (compatible with official vastai v%s)
usage: vastai [global options] command [subcommand] [args...] [flags...]

Global options:
  --api-key KEY    API Key (defaults to ~/.config/vastai/vast_api_key or VAST_API_KEY)
  --url URL        Server REST API URL (default: https://console.vast.ai)
  --raw            Output machine-readable JSON
  --explain        Show underlying API HTTP calls
  --curl           Show equivalent curl command
  --retry RETRY    Set retry limit for API calls (default: 3)
  --version        Show CLI version
  -h, --help       Show this help message

Commands:
  Search:
    search offers [query]        Search GPU offers (flags: --type, -o/--order, --limit, --storage, -n/--no-default)
    search templates [query]     Search public/private templates
    search volumes [query]       Search volume offers
    search benchmarks [query]    Search GPU benchmarks

  Instances:
    show instances               List all your instances (flags: --status, --gpu-name, --label)
    show instance <id>           Show details of a single instance
    create instance <offer-id>   Provision instance (flags: --image, --disk, --ssh, --direct, --jupyter, --label, --bid_price, --onstart-cmd)
    destroy instance <id...>     Destroy instance(s) (-y to skip confirmation)
    start instance <id...>       Start stopped instance(s)
    stop instance <id...>        Stop instance(s)
    reboot instance <id>         Reboot instance
    recycle instance <id>        Recycle instance
    label instance <id> --label <name> Tag instance
    prepay instance <id> <amount> Prepay credits

  SSH & Connectivity:
    ssh-url <id>                 Get ssh:// connection URL
    scp-url <id>                 Get scp:// URL
    attach ssh <id> <key>        Attach SSH public key to instance
    detach ssh <id> <key_id>     Detach SSH key from instance
    show ssh-keys                List registered SSH keys
    create ssh-key [key/file]    Add SSH key (path to .pub or key string)
    delete ssh-key <id>          Remove SSH key

  Logs & Exec:
    logs <id>                    Retrieve container logs (flags: --tail, --filter)
    execute <id> <command>       Execute command on running instance

  Account & Secrets:
    set api-key <key>            Save API key to ~/.config/vastai/vast_api_key
    reset api-key                Reset master API key
    show api-keys                List API keys
    create api-key --name <name> Create restricted API key
    delete api-key <id>          Delete API key
    show user                    Show account profile and credit balance
    show audit-logs              Account action history
    show connections             Cloud storage connections
    show ipaddrs                 IP address history
    show invoices-v1             Invoices and charges history
    show env-vars                List environment variables
    create env-var <key> <val>   Set an environment variable
    delete env-var <key>         Delete an environment variable
    show volumes                 List your volumes
`, config.Version, config.OfficialVersion)
	fmt.Print(helpText)
}

func parseGlobalAndCommandArgs(args []string) (GlobalFlags, []string) {
	var g GlobalFlags
	g.Retry = 3
	var cmdArgs []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--api-key" && i+1 < len(args) {
			g.APIKey = args[i+1]
			i += 2
		} else if strings.HasPrefix(arg, "--api-key=") {
			g.APIKey = strings.TrimPrefix(arg, "--api-key=")
			i++
		} else if arg == "--url" && i+1 < len(args) {
			g.URL = args[i+1]
			i += 2
		} else if strings.HasPrefix(arg, "--url=") {
			g.URL = strings.TrimPrefix(arg, "--url=")
			i++
		} else if arg == "--retry" && i+1 < len(args) {
			if r, err := strconv.Atoi(args[i+1]); err == nil {
				g.Retry = r
			}
			i += 2
		} else if arg == "--raw" {
			g.Raw = true
			i++
		} else if arg == "--explain" {
			g.Explain = true
			i++
		} else if arg == "--curl" {
			g.Curl = true
			i++
		} else if arg == "--version" {
			g.Version = true
			i++
		} else if arg == "-h" || arg == "--help" {
			g.Help = true
			i++
		} else {
			cmdArgs = append(cmdArgs, arg)
			i++
		}
	}

	return g, cmdArgs
}

func getFlagValue(args []string, flagName string) (string, []string) {
	var remaining []string
	val := ""
	for i := 0; i < len(args); i++ {
		if args[i] == flagName && i+1 < len(args) {
			val = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], flagName+"=") {
			val = strings.TrimPrefix(args[i], flagName+"=")
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return val, remaining
}

func getBoolFlag(args []string, flagName string) (bool, []string) {
	var remaining []string
	found := false
	for i := 0; i < len(args); i++ {
		if args[i] == flagName {
			found = true
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return found, remaining
}

func main() {
	rawArgs := os.Args[1:]
	g, cmdArgs := parseGlobalAndCommandArgs(rawArgs)

	if g.Version {
		fmt.Printf("vastai-go %s (compatible with official vastai v%s)\n", config.Version, config.OfficialVersion)
		return
	}

	if g.Help || len(cmdArgs) == 0 {
		printHelp()
		return
	}

	serverURL := config.GetServerURL(g.URL)
	apiKey := config.GetAPIKey(g.APIKey)

	// Command dispatch
	cmd := cmdArgs[0]
	subCmd := ""
	var rest []string
	if len(cmdArgs) > 1 {
		subCmd = cmdArgs[1]
		rest = cmdArgs[2:]
	}

	// 1. set api-key
	if cmd == "set" && subCmd == "api-key" {
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: api-key argument required: vastai set api-key <KEY>")
			os.Exit(1)
		}
		path, err := config.SetAPIKey(rest[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Your api key has been saved in %s\n", path)
		return
	}

	c := client.NewClient(
		client.WithBaseURL(serverURL),
		client.WithAPIKey(apiKey),
		client.WithExplain(g.Explain),
		client.WithCurl(g.Curl),
		client.WithRetry(g.Retry),
	)
	ctx := context.Background()

	// 2. show commands
	if cmd == "show" {
		switch subCmd {
		case "user":
			user, err := api.ShowUser(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.DisplayUser(os.Stdout, user, g.Raw)
			return

		case "instances":
			gpuName, remaining := getFlagValue(rest, "--gpu-name")
			label, remaining := getFlagValue(remaining, "--label")
			statusStr, _ := getFlagValue(remaining, "--status")
			var statuses []string
			if statusStr != "" {
				statuses = strings.Fields(statusStr)
			}

			instances, err := api.ShowInstances(ctx, c, api.ShowInstancesOptions{
				Status:  statuses,
				GPUName: gpuName,
				Label:   label,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.DisplayInstances(os.Stdout, instances, g.Raw)
			return

		case "instance":
			if len(rest) == 0 {
				fmt.Fprintln(os.Stderr, "error: instance ID required: vastai show instance <ID>")
				os.Exit(1)
			}
			id, err := strconv.ParseInt(rest[0], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid instance ID: %s\n", rest[0])
				os.Exit(1)
			}
			inst, err := api.ShowInstance(ctx, c, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.DisplayInstances(os.Stdout, []map[string]interface{}{inst}, g.Raw)
			return

		case "ssh-keys":
			keys, err := api.ShowSSHKeys(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.DisplaySSHKeys(os.Stdout, keys, g.Raw)
			return

		case "env-vars":
			secrets, err := api.ShowEnvVars(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if g.Raw {
				display.PrintJSON(secrets)
			} else {
				for k, v := range secrets {
					fmt.Printf("%s=%s\n", k, v)
				}
			}
			return

		case "api-keys":
			keys, err := api.ShowAPIKeys(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(keys)
			return

		case "audit-logs":
			logs, err := api.ShowAuditLogs(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(logs)
			return

		case "connections":
			conns, err := api.ShowConnections(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(conns)
			return

		case "ipaddrs":
			ips, err := api.ShowIPAddrs(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(ips)
			return

		case "invoices-v1":
			invs, err := api.ShowInvoicesV1(ctx, c, url.Values{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(invs)
			return

		case "volumes":
			vols, err := api.ShowVolumes(ctx, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(vols)
			return
		}
	}

	// 3. search commands
	if cmd == "search" {
		switch subCmd {
		case "offers":
			typeVal, remaining := getFlagValue(rest, "--type")
			orderVal, remaining := getFlagValue(remaining, "-o")
			if orderVal == "" {
				orderVal, remaining = getFlagValue(remaining, "--order")
			}
			limitVal, remaining := getFlagValue(remaining, "--limit")
			storageVal, remaining := getFlagValue(remaining, "--storage")
			noDef, remaining := getBoolFlag(remaining, "-n")
			if !noDef {
				noDef, remaining = getBoolFlag(remaining, "--no-default")
			}

			limit := 0
			if limitVal != "" {
				limit, _ = strconv.Atoi(limitVal)
			}
			storage := 5.0
			if storageVal != "" {
				storage, _ = strconv.ParseFloat(storageVal, 64)
			}

			queryStr := strings.Join(remaining, " ")
			offers, err := api.SearchOffers(ctx, c, api.SearchOffersOptions{
				QueryStr:  queryStr,
				Type:      typeVal,
				Order:     orderVal,
				Limit:     limit,
				Storage:   storage,
				NoDefault: noDef,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.DisplayOffers(os.Stdout, offers, g.Raw)
			return

		case "templates":
			queryStr := strings.Join(rest, " ")
			tmpl, err := api.SearchTemplates(ctx, c, queryStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(tmpl)
			return

		case "benchmarks":
			queryStr := strings.Join(rest, " ")
			bms, err := api.SearchBenchmarks(ctx, c, queryStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(bms)
			return

		case "volumes":
			queryStr := strings.Join(rest, " ")
			vols, err := api.SearchVolumes(ctx, c, queryStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(vols)
			return
		}
	}

	// 4. create instance / create ssh-key / create env-var / create api-key
	if cmd == "create" {
		switch subCmd {
		case "api-key":
			name, remaining := getFlagValue(rest, "--name")
			if name == "" && len(remaining) > 0 {
				name = remaining[0]
			}
			if name == "" {
				name = "default"
			}
			res, err := api.CreateAPIKey(ctx, c, name, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return
		case "instance":
			if len(rest) == 0 {
				fmt.Fprintln(os.Stderr, "error: offer ID required: vastai create instance <OFFER_ID> [flags]")
				os.Exit(1)
			}
			offerID, err := strconv.ParseInt(rest[0], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid offer ID: %s\n", rest[0])
				os.Exit(1)
			}
			args := rest[1:]
			image, args := getFlagValue(args, "--image")
			diskStr, args := getFlagValue(args, "--disk")
			label, args := getFlagValue(args, "--label")
			onstart, args := getFlagValue(args, "--onstart-cmd")
			bidPriceStr, args := getFlagValue(args, "--bid_price")
			templateHash, args := getFlagValue(args, "--template_hash")
			ssh, args := getBoolFlag(args, "--ssh")
			direct, args := getBoolFlag(args, "--direct")
			jupyter, args := getBoolFlag(args, "--jupyter")
			cancelUnavail, _ := getBoolFlag(args, "--cancel-unavail")

			disk := 10.0
			if diskStr != "" {
				disk, _ = strconv.ParseFloat(diskStr, 64)
			}
			var bidPrice *float64
			if bidPriceStr != "" {
				if bp, err := strconv.ParseFloat(bidPriceStr, 64); err == nil {
					bidPrice = &bp
				}
			}

			res, err := api.CreateInstance(ctx, c, api.CreateInstanceOptions{
				OfferID:       offerID,
				Image:         image,
				Disk:          disk,
				SSH:           ssh,
				Direct:        direct,
				Jupyter:       jupyter,
				Label:         label,
				OnstartCmd:    onstart,
				BidPrice:      bidPrice,
				TemplateHash:  templateHash,
				CancelUnavail: cancelUnavail,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return

		case "ssh-key":
			keyStr := ""
			if len(rest) > 0 {
				keyStr = rest[0]
			}
			res, err := api.CreateSSHKey(ctx, c, keyStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return

		case "env-var":
			if len(rest) < 2 {
				fmt.Fprintln(os.Stderr, "error: key and value required: vastai create env-var <KEY> <VALUE>")
				os.Exit(1)
			}
			res, err := api.CreateEnvVar(ctx, c, rest[0], rest[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return
		}
	}

	// 5. destroy instance
	if cmd == "destroy" && (subCmd == "instance" || subCmd == "instances") {
		yes, remaining := getBoolFlag(rest, "-y")
		if !yes {
			yes, remaining = getBoolFlag(remaining, "--yes")
		}
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "error: instance ID(s) required: vastai destroy instance <ID...> [-y]")
			os.Exit(1)
		}
		var ids []int64
		for _, arg := range remaining {
			if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}

		if !yes {
			fmt.Printf("Are you sure you want to destroy instance(s) %v? (y/N): ", ids)
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(strings.ToLower(text))
			if text != "y" && text != "yes" {
				fmt.Println("Cancelled.")
				return
			}
		}

		res, err := api.DestroyInstance(ctx, c, ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	// 6. start / stop / reboot / recycle instance
	if cmd == "start" && subCmd == "instance" {
		var ids []int64
		for _, arg := range rest {
			if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		res, err := api.StartInstance(ctx, c, ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	if cmd == "stop" && subCmd == "instance" {
		var ids []int64
		for _, arg := range rest {
			if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		res, err := api.StopInstance(ctx, c, ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	if cmd == "reboot" && subCmd == "instance" {
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: instance ID required")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		res, err := api.RebootInstance(ctx, c, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	if cmd == "recycle" && subCmd == "instance" {
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: instance ID required")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		res, err := api.RecycleInstance(ctx, c, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	// 7. label instance
	if cmd == "label" && subCmd == "instance" {
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: instance ID required: vastai label instance <ID> --label <NAME>")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		lbl, _ := getFlagValue(rest[1:], "--label")
		res, err := api.LabelInstance(ctx, c, id, lbl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	// 8. prepay instance
	if cmd == "prepay" && subCmd == "instance" {
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "error: instance ID and amount required: vastai prepay instance <ID> <AMOUNT>")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		amount, _ := strconv.ParseFloat(rest[1], 64)
		res, err := api.PrepayInstance(ctx, c, id, amount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	// 9. ssh-url / scp-url
	if cmd == "ssh-url" || (cmd == "ssh" && subCmd == "url") {
		args := rest
		if cmd == "ssh-url" {
			args = append([]string{subCmd}, rest...)
		}
		if len(args) == 0 || args[0] == "" {
			fmt.Fprintln(os.Stderr, "error: instance ID required: vastai ssh-url <ID>")
			os.Exit(1)
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid instance ID: %s\n", args[0])
			os.Exit(1)
		}
		urlStr, err := api.GetSSHURL(ctx, c, id, "ssh://")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(urlStr)
		return
	}

	if cmd == "scp-url" || (cmd == "scp" && subCmd == "url") {
		args := rest
		if cmd == "scp-url" {
			args = append([]string{subCmd}, rest...)
		}
		if len(args) == 0 || args[0] == "" {
			fmt.Fprintln(os.Stderr, "error: instance ID required: vastai scp-url <ID>")
			os.Exit(1)
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid instance ID: %s\n", args[0])
			os.Exit(1)
		}
		urlStr, err := api.GetSSHURL(ctx, c, id, "scp://")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(urlStr)
		return
	}

	// 10. logs
	if cmd == "logs" {
		idStr := subCmd
		args := rest
		tailStr, args := getFlagValue(args, "--tail")
		filterStr, _ := getFlagValue(args, "--filter")
		if idStr == "" {
			fmt.Fprintln(os.Stderr, "error: instance ID required: vastai logs <ID>")
			os.Exit(1)
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid instance ID: %s\n", idStr)
			os.Exit(1)
		}
		tail := 0
		if tailStr != "" {
			tail, _ = strconv.Atoi(tailStr)
		}
		logText, err := api.Logs(ctx, c, id, tail, filterStr, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(logText)
		return
	}

	// 11. execute
	if cmd == "execute" {
		idStr := subCmd
		if idStr == "" || len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: instance ID and command required: vastai execute <ID> <COMMAND>")
			os.Exit(1)
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid instance ID: %s\n", idStr)
			os.Exit(1)
		}
		command := strings.Join(rest, " ")
		out, err := api.Execute(ctx, c, id, command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	// 12. delete commands
	if cmd == "delete" {
		switch subCmd {
		case "ssh-key":
			if len(rest) == 0 {
				fmt.Fprintln(os.Stderr, "error: ssh-key ID required: vastai delete ssh-key <ID>")
				os.Exit(1)
			}
			id, _ := strconv.ParseInt(rest[0], 10, 64)
			res, err := api.DeleteSSHKey(ctx, c, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return

		case "api-key":
			if len(rest) == 0 {
				fmt.Fprintln(os.Stderr, "error: api-key ID required: vastai delete api-key <ID>")
				os.Exit(1)
			}
			id, _ := strconv.ParseInt(rest[0], 10, 64)
			res, err := api.DeleteAPIKey(ctx, c, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return

		case "env-var":
			if len(rest) == 0 {
				fmt.Fprintln(os.Stderr, "error: env-var name required: vastai delete env-var <KEY>")
				os.Exit(1)
			}
			res, err := api.DeleteEnvVar(ctx, c, rest[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			display.PrintJSON(res)
			return
		}
	}

	// 13. attach / detach ssh
	if cmd == "attach" && subCmd == "ssh" {
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "error: instance ID and ssh key required: vastai attach ssh <ID> <KEY>")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		res, err := api.AttachSSH(ctx, c, id, rest[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	if cmd == "detach" && subCmd == "ssh" {
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "error: instance ID and ssh key ID required: vastai detach ssh <ID> <KEY_ID>")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(rest[0], 10, 64)
		res, err := api.DetachSSH(ctx, c, id, rest[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	// 14. reset api-key
	if cmd == "reset" && subCmd == "api-key" {
		res, err := api.ResetAPIKey(ctx, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		display.PrintJSON(res)
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command: %s %s\nRun 'vastai --help' for usage.\n", cmd, subCmd)
	os.Exit(1)
}
