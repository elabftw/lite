/*
 * This file is part of eLabFTW lite.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * File handling specific to iOS and Android
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// On mobile, the system file picker hands us a copy of the chosen file inside
// the app sandbox (iOS: tmp/<bundle>-Inbox, Android: cache/wails-picker). We
// track those copies so they can be removed once imported, instead of leaving
// plaintext duplicates of encrypted uploads lying around.
var (
	pickedFilesMu sync.Mutex
	pickedFiles   = map[string]struct{}{}
)

func isMobile() bool {
	return runtime.GOOS == "ios" || runtime.GOOS == "android"
}

// rememberPickedFile records a path returned by the file picker on mobile.
func rememberPickedFile(path string) {
	if !isMobile() || path == "" {
		return
	}
	pickedFilesMu.Lock()
	pickedFiles[path] = struct{}{}
	pickedFilesMu.Unlock()
}

// discardPickedFile deletes a picker copy. Only paths previously returned by
// SelectFile are ever removed, so a caller can't use it to delete arbitrary files.
func discardPickedFile(path string) {
	pickedFilesMu.Lock()
	_, ok := pickedFiles[path]
	delete(pickedFiles, path)
	pickedFilesMu.Unlock()
	if !ok {
		return
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove picked file copy: %v", err)
	}
	// The picker puts copies in their own folder: drop it if it is now empty.
	_ = os.Remove(filepath.Dir(path))
}

// exportDir holds decrypted files handed to the share sheet. It lives under the
// app root so Android's FileProvider can expose it (see res/xml/file_paths.xml).
func exportDir() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "exports"), nil
}

// clearExports removes previously shared plaintext files.
func clearExports() {
	dir, err := exportDir()
	if err != nil {
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("clear exports: %v", err)
	}
}

// shareDecryptedFile writes content to the private export folder and opens the
// native share sheet, from which the user can save it to Files, AirDrop it, etc.
// Mobile platforms have no save dialog, so this replaces it there.
// Only the latest export is kept: earlier ones are cleared first.
func shareDecryptedFile(name string, content []byte) (string, error) {
	dir, err := exportDir()
	if err != nil {
		return "", err
	}
	clearExports()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}
	sub, err := os.MkdirTemp(dir, "share-")
	if err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}

	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "upload"
	}
	path := filepath.Join(sub, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write export: %w", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(name))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	payload, err := json.Marshal(map[string]string{
		"url":  (&url.URL{Scheme: "file", Path: path}).String(),
		"mime": mimeType,
	})
	if err != nil {
		return "", err
	}
	application.Mobile.Share(string(payload))

	return path, nil
}
