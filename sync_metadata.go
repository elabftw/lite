/*
 * This file is part of eLabFTW lite.
 * SPDX-License-Identifier: GPL-3.0-or-later
 */
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Resolve reference fields once during pull/import, not when opening an offline entry.
// Missing or inaccessible references retain their original IDs.
func (a *App) resolveMetadataLabels(profileUUID string, instanceID int64, raw json.RawMessage) map[string]string {
	var encoded string
	if json.Unmarshal(raw, &encoded) == nil {
		raw = json.RawMessage(encoded)
	}
	var metadata struct {
		ExtraFields map[string]struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"extra_fields"`
	}
	if json.Unmarshal(raw, &metadata) != nil {
		return nil
	}
	labels := make(map[string]string)
	seen := make(map[string]bool)
	for _, field := range metadata.ExtraFields {
		switch field.Type {
		case "users", "items", "experiments", "compounds":
		default:
			continue
		}
		var values []json.RawMessage
		if json.Unmarshal(field.Value, &values) != nil {
			values = []json.RawMessage{field.Value}
		}
		for _, value := range values {
			idText := string(value)
			var quoted string
			if json.Unmarshal(value, &quoted) == nil {
				idText = quoted
			}
			id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			key := field.Type + ":" + strings.TrimSpace(idText)
			if seen[key] {
				continue
			}
			seen[key] = true
			resp, err := a.elabftwRequest(profileUUID, instanceID, http.MethodGet, fmt.Sprintf("/%s/%d", field.Type, id), nil, false)
			if err != nil {
				continue
			}
			var target struct {
				Fullname  string `json:"fullname"`
				Title     string `json:"title"`
				Name      string `json:"name"`
				IUPACName string `json:"iupac_name"`
			}
			if decodeElabftwJSONResponse(resp, &target) != nil {
				continue
			}
			label := target.Title
			switch field.Type {
			case "users":
				label = target.Fullname
			case "compounds":
				label = target.Name
				if strings.TrimSpace(label) == "" {
					label = target.IUPACName
				}
			}
			if strings.TrimSpace(label) != "" {
				labels[key] = label
			}
		}
	}
	return labels
}
