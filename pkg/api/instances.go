package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/takayoshi/vast-cli/pkg/client"
)

// ShowInstancesOptions filters and sorts instances
type ShowInstancesOptions struct {
	Status  []string
	GPUName string
	Label   string
}

// ShowInstances fetches all user instances by paginating through /api/v1/instances/
func ShowInstances(ctx context.Context, c *client.Client, opts ShowInstancesOptions) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	afterToken := ""

	selectFilters := make(map[string]interface{})
	if len(opts.Status) > 0 {
		selectFilters["actual_status"] = map[string]interface{}{"in": opts.Status}
	}
	if opts.GPUName != "" {
		selectFilters["gpu_name"] = map[string]interface{}{"eq": opts.GPUName}
	}
	if opts.Label != "" {
		selectFilters["label"] = map[string]interface{}{"eq": opts.Label}
	}

	for {
		queryParams := url.Values{}
		queryParams.Set("limit", "25")
		if afterToken != "" {
			queryParams.Set("after_token", afterToken)
		}
		if len(selectFilters) > 0 {
			fJSON, _ := json.Marshal(selectFilters)
			queryParams.Set("select_filters", string(fJSON))
		}
		orderBy := []map[string]string{{"col": "id", "dir": "asc"}}
		oJSON, _ := json.Marshal(orderBy)
		queryParams.Set("order_by", string(oJSON))

		data, err := c.Get(ctx, "/api/v1/instances/", queryParams)
		if err != nil {
			// Fallback to /api/v0/instances/ if v1 fails
			return fallbackShowInstancesV0(ctx, c)
		}

		var pageResp struct {
			Instances []map[string]interface{} `json:"instances"`
			NextToken string                   `json:"next_token"`
		}
		if err := json.Unmarshal(data, &pageResp); err != nil {
			return nil, fmt.Errorf("failed to parse instances: %w", err)
		}

		rows = append(rows, pageResp.Instances...)
		if pageResp.NextToken == "" {
			break
		}
		afterToken = pageResp.NextToken
	}

	return rows, nil
}

func fallbackShowInstancesV0(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/instances/", url.Values{"owner": []string{"me"}})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Instances, nil
}

// ShowInstance retrieves a single instance by ID
func ShowInstance(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/instances/%d/", id), url.Values{"owner": []string{"me"}})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Instances map[string]interface{} `json:"instances"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse instance response: %w", err)
	}
	return resp.Instances, nil
}

// CreateInstanceOptions holds options for launching an instance
type CreateInstanceOptions struct {
	OfferID      int64
	Image        string
	Disk         float64
	SSH          bool
	Jupyter      bool
	Direct       bool
	Label        string
	Env          map[string]string
	OnstartCmd   string
	BidPrice     *float64
	TemplateHash string
	CancelUnavail bool
}

// CreateInstance provisions an instance from an offer
func CreateInstance(ctx context.Context, c *client.Client, opts CreateInstanceOptions) (map[string]interface{}, error) {
	runtype := "ssh_proxy"
	if opts.Jupyter {
		if opts.Direct {
			runtype = "jupyter_direc ssh_direc ssh_proxy"
		} else {
			runtype = "jupyter_proxy ssh_proxy"
		}
	} else if opts.SSH || opts.Direct {
		if opts.Direct {
			runtype = "ssh_direc ssh_proxy"
		} else {
			runtype = "ssh_proxy"
		}
	}

	disk := opts.Disk
	if disk <= 0 {
		disk = 10.0
	}

	payload := map[string]interface{}{
		"client_id":      "me",
		"image":          opts.Image,
		"disk":           disk,
		"runtype":        runtype,
		"cancel_unavail": opts.CancelUnavail,
	}

	if opts.Label != "" {
		payload["label"] = opts.Label
	}
	if opts.OnstartCmd != "" {
		payload["onstart"] = opts.OnstartCmd
	}
	if opts.TemplateHash != "" {
		payload["template_hash_id"] = opts.TemplateHash
	}
	if opts.BidPrice != nil {
		payload["price"] = *opts.BidPrice
	}
	if len(opts.Env) > 0 {
		payload["env"] = opts.Env
	}

	data, err := c.Put(ctx, fmt.Sprintf("/asks/%d/", opts.OfferID), url.Values{}, payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create instance response: %w", err)
	}
	return result, nil
}

// DestroyInstance destroys one or more instances
func DestroyInstance(ctx context.Context, c *client.Client, ids []int64) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no instance IDs specified")
	}

	var data []byte
	var err error
	if len(ids) == 1 {
		data, err = c.Delete(ctx, fmt.Sprintf("/instances/%d/", ids[0]), url.Values{}, map[string]interface{}{})
	} else {
		data, err = c.Delete(ctx, "/instances/", url.Values{}, map[string]interface{}{"instance_ids": ids})
	}
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// StartInstance starts one or more stopped instances
func StartInstance(ctx context.Context, c *client.Client, ids []int64) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no instance IDs specified")
	}

	var data []byte
	var err error
	if len(ids) == 1 {
		data, err = c.Put(ctx, fmt.Sprintf("/instances/%d/", ids[0]), url.Values{}, map[string]interface{}{"state": "running"})
	} else {
		data, err = c.Put(ctx, "/instances/", url.Values{}, map[string]interface{}{"ids": ids, "state": "running"})
	}
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// StopInstance stops one or more running instances
func StopInstance(ctx context.Context, c *client.Client, ids []int64) (map[string]interface{}, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no instance IDs specified")
	}

	var data []byte
	var err error
	if len(ids) == 1 {
		data, err = c.Put(ctx, fmt.Sprintf("/instances/%d/", ids[0]), url.Values{}, map[string]interface{}{"state": "stopped"})
	} else {
		data, err = c.Put(ctx, "/instances/", url.Values{}, map[string]interface{}{"ids": ids, "state": "stopped"})
	}
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// RebootInstance reboots an instance
func RebootInstance(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Put(ctx, fmt.Sprintf("/instances/reboot/%d/", id), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// RecycleInstance destroys and recreates an instance
func RecycleInstance(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Put(ctx, fmt.Sprintf("/instances/recycle/%d/", id), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// LabelInstance sets a tag/label on an instance
func LabelInstance(ctx context.Context, c *client.Client, id int64, label string) (map[string]interface{}, error) {
	data, err := c.Put(ctx, fmt.Sprintf("/instances/%d/", id), url.Values{}, map[string]interface{}{"label": label})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// PrepayInstance deposits credits into a reserved instance
func PrepayInstance(ctx context.Context, c *client.Client, id int64, amount float64) (map[string]interface{}, error) {
	data, err := c.Put(ctx, fmt.Sprintf("/instances/prepay/%d/", id), url.Values{}, map[string]interface{}{"amount": amount})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// GetSSHURL generates ssh:// or scp:// connection URL for an instance
func GetSSHURL(ctx context.Context, c *client.Client, id int64, protocol string) (string, error) {
	inst, err := ShowInstance(ctx, c, id)
	if err != nil {
		return "", err
	}
	if inst == nil {
		return "", fmt.Errorf("instance %d not found", id)
	}

	var ipaddr string
	var port int64 = -1

	// Check if direct 22/tcp port mapping exists
	if ports, ok := inst["ports"].(map[string]interface{}); ok {
		if p22, ok := ports["22/tcp"].([]interface{}); ok && len(p22) > 0 {
			if entry, ok := p22[0].(map[string]interface{}); ok {
				if hostPortStr, ok := entry["HostPort"].(string); ok {
					if p, err := strconv.ParseInt(hostPortStr, 10, 64); err == nil {
						port = p
						if pubIP, ok := inst["public_ipaddr"].(string); ok {
							ipaddr = pubIP
						}
					}
				}
			}
		}
	}

	// Fallback to ssh_host / ssh_port (SSH proxy)
	if port <= 0 {
		if host, ok := inst["ssh_host"].(string); ok && host != "" {
			ipaddr = host
		}
		if p, ok := inst["ssh_port"].(float64); ok && p > 0 {
			port = int64(p)
			runtype, _ := inst["image_runtype"].(string)
			if strings.Contains(runtype, "jupyter") {
				port++
			}
		}
	}

	if port <= 0 || ipaddr == "" {
		return "", fmt.Errorf("ssh port or host not found for instance %d (actual status: %v)", id, inst["actual_status"])
	}

	return fmt.Sprintf("%sroot@%s:%d", protocol, ipaddr, port), nil
}

// Logs retrieves logs of an instance
func Logs(ctx context.Context, c *client.Client, id int64, tail int, filter string, daemonLogs bool) (string, error) {
	body := make(map[string]interface{})
	if tail > 0 {
		body["tail"] = tail
	}
	if filter != "" {
		body["filter"] = filter
	}
	if daemonLogs {
		body["daemon_logs"] = "true"
	}

	data, err := c.Put(ctx, fmt.Sprintf("/instances/request_logs/%d/", id), url.Values{}, body)
	if err != nil {
		return "", err
	}

	var res struct {
		ResultURL string `json:"result_url"`
		Msg       string `json:"msg"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return string(data), nil
	}

	if res.ResultURL == "" {
		if res.Msg != "" {
			return res.Msg, nil
		}
		return string(data), nil
	}

	return c.PollResultURL(ctx, res.ResultURL, 30, 300*time.Millisecond)
}

// Execute executes a command on an instance and returns output
func Execute(ctx context.Context, c *client.Client, id int64, command string) (string, error) {
	data, err := c.Put(ctx, fmt.Sprintf("/instances/command/%d/", id), url.Values{}, map[string]string{"command": command})
	if err != nil {
		return "", err
	}

	var res struct {
		ResultURL string `json:"result_url"`
		Msg       string `json:"msg"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return string(data), nil
	}

	if res.ResultURL == "" {
		if res.Msg != "" {
			return res.Msg, nil
		}
		return string(data), nil
	}

	return c.PollResultURL(ctx, res.ResultURL, 30, 300*time.Millisecond)
}
