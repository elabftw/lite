/*
 * This file is part of eLabFTW lite.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */
package main

// AppVersion is set from the release tag at build time.
var AppVersion = "dev"

// GetAppVersion returns the version embedded at build time.
func (a *App) GetAppVersion() string {
	return AppVersion
}
