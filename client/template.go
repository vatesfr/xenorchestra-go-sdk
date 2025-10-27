package client

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type TemplateBoot struct {
	Firmware string `json:"firmware"`
	Order    string `json:"order"`
}

type TemplateDisk struct {
	Bootable bool   `json:"bootable"`
	Device   string `json:"device"`
	Size     int    `json:"size"`
	Type     string `json:"type"`
	SR       string `json:"SR"`
}

type TemplateInfo struct {
	Arch  string         `json:"arch"`
	Disks []TemplateDisk `json:"disks"`
}

type Template struct {
	Id           string       `json:"id"`
	Uuid         string       `json:"uuid"`
	Boot         TemplateBoot `json:"boot"`
	NameLabel    string       `json:"name_label"`
	PoolId       string       `json:"$poolId"`
	TemplateInfo TemplateInfo `json:"template_info"`
	// Array of VDI ids
	VBDs []string `json:"$VBDs"`
}

func (t Template) Compare(obj interface{}) bool {
	other := obj.(Template)

	if t.Id == other.Id {
		return true
	}

	labelsMatch := t.NameLabel == other.NameLabel

	if t.PoolId == "" && labelsMatch {
		return true
	} else if t.PoolId == other.PoolId && labelsMatch {
		return true
	}
	return false
}

func (t Template) isDiskTemplate() bool {
	if len(t.VBDs) != 0 && t.NameLabel != "Other install media" {
		return true
	}

	return false
}

func (c *Client) GetTemplate(template Template) ([]Template, error) {
	obj, err := c.FindFromGetAllObjects(template)
	var templates []Template
	if err != nil {
		return templates, err
	}

	templates, ok := obj.([]Template)

	if !ok {
		return templates, errors.New("failed to coerce response into Template slice")
	}

	return templates, nil
}

func FindTemplateForTests(template *Template, poolId, templateEnvVar string) {
	var found bool
	templateName, found := os.LookupEnv(templateEnvVar)
	if !found {
		slog.Error("The environment variable must be set for the tests", "name", templateEnvVar)
		os.Exit(-1)
	}

	c, err := NewClient(GetConfigFromEnv())
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(-1)
	}

	templates, err := c.GetTemplate(Template{
		NameLabel: templateName,
		PoolId:    poolId,
	})

	if err != nil {
		slog.Error("failed to find templates", "error", err)
		os.Exit(-1)
	}

	l := len(templates)
	if l != 1 {
		slog.Error(fmt.Sprintf("found %d templates when expected to find 1. templates found: %v\n", l, templates))
		os.Exit(-1)
	}
	*template = templates[0]
}

func (t *Template) getDiskCount() int {
	return len(t.VBDs)
}

// GetTemplateVBDs retrieves all VBDs for a given template and returns them as a map
// where the key is the VBD's position.
func (c *Client) GetTemplateVBDs(template Template) (map[string]VBD, error) {
	var response map[string]VBD
	err := c.GetAllObjectsOfType(VBD{}, &response)
	if err != nil {
		slog.Error("failed to get template VBDs", "error", err)
		return nil, err
	}

	vbds := make(map[string]VBD, len(template.VBDs))
	for _, vbd := range response {
		if vbd.VmId != template.Id {
			continue
		}
		vbds[vbd.Position] = vbd
	}
	return vbds, nil
}
