// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import "github.com/pterm/pterm"

const repoUTM = "hypersdk"

func platformURL() string {
	return "https://zyvor.dev/?utm_source=github&utm_medium=" + repoUTM
}

func contactURL() string {
	return "https://zyvor.dev/contact?utm_source=github&utm_medium=" + repoUTM
}

func showEnterpriseCTA() {
	pterm.Println()
	pterm.DefaultBox.WithTitle("Enterprise & HyperSDK Platform").
		WithTitleTopCenter().
		Println(
			"Community Edition on GitHub · Production via Zyvor AI Labs\n\n" +
				"• Multi-cloud export at scale · VMware exit · air-gapped\n" +
				"• Suite: HyperSDK · hyper2kvm · GuestKit · v9s · PacketWolf\n\n" +
				platformURL() + "\n" +
				contactURL() + "\n" +
				"sales@zyvor.dev · info@zyvor.dev",
		)
}
