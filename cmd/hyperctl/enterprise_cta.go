// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import "github.com/pterm/pterm"

const repoUTM = "hypersdk_repo"

func platformURL() string {
	return "https://zyvor.dev/?utm_source=github&utm_medium=" + repoUTM
}

func contactURL() string {
	return "https://zyvor.dev/contact?utm_source=github&utm_medium=" + repoUTM
}

func demoURL() string {
	return "https://zyvor.dev/demo?utm_source=github&utm_medium=" + repoUTM
}

func showEnterpriseCTA() {
	pterm.Println()
	pterm.DefaultBox.WithTitle("Enterprise & HyperSDK Platform").
		WithTitleTopCenter().
		Println(
			"Community Edition on GitHub · Production via Zyvor AI Labs\n\n" +
				"▶ Watch migration demo (video + live dashboard):\n" +
				demoURL() + "\n\n" +
				"• Multi-cloud export · VMware exit · air-gapped\n" +
				"• Suite: HyperSDK · hyper2kvm · GuestKit · v9s · PacketWolf\n\n" +
				contactURL() + " (guided demo)\n" +
				"sales@zyvor.dev · info@zyvor.dev",
		)
}
