package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintJSON outputs data as indented JSON
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Offer represents a single GPU offer row from Vast.ai
type Offer map[string]interface{}

func getFloat(m map[string]interface{}, key string) float64 {
	val, ok := m[key]
	if !ok || val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int64 {
	val, ok := m[key]
	if !ok || val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	val, ok := m[key]
	if !ok || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// DisplayOffers prints offers either as raw JSON or as formatted multi-block tables
func DisplayOffers(w io.Writer, offers []map[string]interface{}, raw bool) error {
	if raw {
		return PrintJSON(offers)
	}

	if len(offers) == 0 {
		fmt.Fprintln(w, "No offers found matching query.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Block 1: Hardware specs
	fmt.Fprintln(tw, "  #\tID\tCUDA\tN\tModel\tPCIE\tcpu_ghz\tvCPUs\tRAM(GB)\tVRAM(GB)")
	for i, o := range offers {
		id := getInt(o, "id")
		cuda := getFloat(o, "cuda_max_good")
		numGpus := getInt(o, "num_gpus")
		gpuName := strings.ReplaceAll(getString(o, "gpu_name"), " ", "_")
		pcie := getFloat(o, "pcie_bw")
		cpuGhz := getFloat(o, "cpu_ghz")
		cpuCores := getFloat(o, "cpu_cores_effective")
		if cpuCores == 0 {
			cpuCores = float64(getInt(o, "cpu_cores"))
		}
		ram := getFloat(o, "cpu_ram") / 1000.0
		vram := getFloat(o, "gpu_ram") / 1000.0

		fmt.Fprintf(tw, "  %d\t%d\t%.1f\t%dx\t%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n",
			i+1, id, cuda, numGpus, gpuName, pcie, cpuGhz, cpuCores, ram, vram)
	}
	tw.Flush()

	fmt.Fprintln(w)

	// Block 2: Performance, Price, Network
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  #\tDisk(GB)\t$/hr\tDLP\tDLP/$\tscore\tNV Driver\tNet_up(M)\tNet_down(M)\tR(%)")
	for i, o := range offers {
		disk := getFloat(o, "disk_space")
		dph := getFloat(o, "dph_total")
		dlp := getFloat(o, "dlperf")
		dlpUsd := getFloat(o, "dlperf_per_dphtotal")
		score := getFloat(o, "score")
		driver := getString(o, "driver_version")
		netUp := getFloat(o, "inet_up")
		netDown := getFloat(o, "inet_down")
		rel := getFloat(o, "reliability") * 100.0

		fmt.Fprintf(tw, "  %d\t%.0f\t%.4f\t%.1f\t%.2f\t%.1f\t%s\t%.1f\t%.1f\t%.1f\n",
			i+1, disk, dph, dlp, dlpUsd, score, driver, netUp, netDown, rel)
	}
	tw.Flush()

	fmt.Fprintln(w)

	// Block 3: Machine & Location
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  #\tMax_Days\tmach_id\tstatus\thost_id\tports\tcountry")
	for i, o := range offers {
		durationSec := getFloat(o, "duration")
		maxDays := durationSec / 86400.0
		machID := getInt(o, "machine_id")
		status := getString(o, "verification")
		if status == "" {
			status = "verified"
		}
		hostID := getInt(o, "host_id")
		ports := getInt(o, "direct_port_count")
		country := strings.ReplaceAll(getString(o, "geolocation"), " ", "_")

		fmt.Fprintf(tw, "  %d\t%.1f\t%d\t%s\t%d\t%d\t%s\n",
			i+1, maxDays, machID, status, hostID, ports, country)
	}
	tw.Flush()

	return nil
}

// DisplayInstances prints instances list
func DisplayInstances(w io.Writer, instances []map[string]interface{}, raw bool) error {
	if raw {
		return PrintJSON(instances)
	}

	if len(instances) == 0 {
		fmt.Fprintln(w, "No instances found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tStatus\tActual\tModel\tNum\t$/hr\tImage\tSSH URL\tLabel")
	for _, inst := range instances {
		id := getInt(inst, "id")
		status := getString(inst, "intended_status")
		actual := getString(inst, "actual_status")
		model := getString(inst, "gpu_name")
		num := getInt(inst, "num_gpus")
		dph := getFloat(inst, "dph_total")
		image := getString(inst, "image_runtype")
		if curImage := getString(inst, "image_uuid"); curImage != "" {
			image = curImage
		}
		if len(image) > 24 {
			image = image[:21] + "..."
		}
		label := getString(inst, "label")

		// Calculate SSH URL if available
		sshURL := "-"
		sshHost := getString(inst, "ssh_host")
		sshPort := getInt(inst, "ssh_port")
		if sshHost != "" && sshPort > 0 {
			sshURL = fmt.Sprintf("ssh://root@%s:%d", sshHost, sshPort)
		}

		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%dx\t$%.4f\t%s\t%s\t%s\n",
			id, status, actual, model, num, dph, image, sshURL, label)
	}
	tw.Flush()
	return nil
}

// DisplayUser prints user account details
func DisplayUser(w io.Writer, u map[string]interface{}, raw bool) error {
	if raw {
		return PrintJSON(u)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "User ID:\t%d\n", getInt(u, "id"))
	fmt.Fprintf(tw, "Username:\t%s\n", getString(u, "username"))
	fmt.Fprintf(tw, "Email:\t%s\n", getString(u, "email"))
	fmt.Fprintf(tw, "Credit Balance:\t$%.2f\n", getFloat(u, "credit"))
	fmt.Fprintf(tw, "Balance:\t$%.2f\n", getFloat(u, "balance"))
	fmt.Fprintf(tw, "Type:\t%s\n", getString(u, "type"))
	fmt.Fprintf(tw, "Can Rent:\t%v\n", u["can_rent"])
	fmt.Fprintf(tw, "SSH Key Present:\t%v\n", u["has_ssh_key"])
	tw.Flush()
	return nil
}

// DisplaySSHKeys prints list of SSH keys
func DisplaySSHKeys(w io.Writer, keys []map[string]interface{}, raw bool) error {
	if raw {
		return PrintJSON(keys)
	}

	if len(keys) == 0 {
		fmt.Fprintln(w, "No SSH keys registered.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tType\tFingerprint\tKey")
	for _, k := range keys {
		id := getInt(k, "id")
		keyStr := getString(k, "ssh_key")
		parts := strings.Fields(keyStr)
		keyType := "-"
		keyPreview := keyStr
		if len(parts) >= 2 {
			keyType = parts[0]
			keyPreview = parts[1]
			if len(keyPreview) > 30 {
				keyPreview = keyPreview[:15] + "..." + keyPreview[len(keyPreview)-10:]
			}
		}
		fp := getString(k, "fingerprint")
		if fp == "" {
			fp = "-"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", id, keyType, fp, keyPreview)
	}
	tw.Flush()
	return nil
}
