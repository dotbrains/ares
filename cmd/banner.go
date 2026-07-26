package cmd

import (
	cryptorand "crypto/rand"
	"math/big"

	figure "github.com/common-nighthawk/go-figure"
	"github.com/spf13/cobra"
)

var bannerFonts = []string{
	"slant",
	"small",
	"shadow",
	"doom",
	"standard",
	"straight",
	"digital",
	"mini",
}

func printBanner(cmd *cobra.Command) {
	banner := figure.NewFigure("ares", randomBannerFont(), true)
	cmd.Println(banner.String())
}

func randomBannerFont() string {
	index, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(bannerFonts))))
	if err != nil {
		return bannerFonts[0]
	}
	return bannerFonts[index.Int64()]
}
