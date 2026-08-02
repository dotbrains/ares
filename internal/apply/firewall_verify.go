package apply

import (
	"context"
	"fmt"
	"strings"
)

var webProfileVerifiers = map[string]func(*Context, string){
	"firewalld": (*Context).verifyFirewalldWebProfile,
	"nftables":  (*Context).verifyNftablesWebProfile,
	"ufw":       (*Context).verifyUFWWebProfile,
}

func (ctx *Context) verifyUFW(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"ufw", "status"}, ctx.Plan.Host.SSHPort+"/tcp")
}

func (ctx *Context) verifyFirewalld(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"firewall-cmd", "--list-all"}, ctx.Plan.Host.SSHPort+"/tcp")
}

func (ctx *Context) verifyNftables(pluginID string) {
	ctx.verifyPath(pluginID, "/etc/nftables.conf")
	ctx.verifyCommandContains(pluginID, []string{"nft", "list", "ruleset"}, "dport "+ctx.Plan.Host.SSHPort)
}

func (ctx *Context) verifyWebProfile(pluginID string) {
	if verify, ok := webProfileVerifiers[ctx.Plan.Host.FirewallBackend]; ok {
		verify(ctx, pluginID)
		return
	}
	ctx.Result.Failed = append(ctx.Result.Failed, fmt.Sprintf("%s: unsupported firewall backend %q for verification", pluginID, ctx.Plan.Host.FirewallBackend))
}

func (ctx *Context) verifyFirewalldWebProfile(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"firewall-cmd", "--list-all"}, "services:")
	ctx.verifyCommandContains(pluginID, []string{"firewall-cmd", "--list-all"}, "http")
	ctx.verifyCommandContains(pluginID, []string{"firewall-cmd", "--list-all"}, "https")
}

func (ctx *Context) verifyNftablesWebProfile(pluginID string) {
	ctx.verifyPath(pluginID, "/etc/nftables.conf")
	ctx.verifyCommandContains(pluginID, []string{"nft", "list", "ruleset"}, "dport 80")
	ctx.verifyCommandContains(pluginID, []string{"nft", "list", "ruleset"}, "dport 443")
}

func (ctx *Context) verifyUFWWebProfile(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"ufw", "status"}, "80/tcp")
	ctx.verifyCommandContains(pluginID, []string{"ufw", "status"}, "443/tcp")
}

func (ctx *Context) verifyCommandContains(pluginID string, command []string, want string) {
	if ctx.Options.Root != "" {
		ctx.Result.Verified = append(ctx.Result.Verified, pluginID+": would verify with "+strings.Join(command, " "))
		return
	}
	output, err := ctx.Options.Runner.Run(context.Background(), command[0], command[1:]...)
	if err != nil {
		ctx.Result.Failed = append(ctx.Result.Failed, pluginID+": "+err.Error())
		return
	}
	if !strings.Contains(output, want) {
		ctx.Result.Failed = append(ctx.Result.Failed, pluginID+": verification output missing "+want)
		return
	}
	ctx.Result.Verified = append(ctx.Result.Verified, pluginID+": verified "+strings.Join(command, " "))
}
