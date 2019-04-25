package cmd

import (
	"fmt"
	"github.com/ontio/ontology/blockrelayer"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/urfave/cli"
)

var RevertCommand = cli.Command{
	Name:  "revert",
	Usage: "revert current info to height",
	Action: revertToHeight,
	Flags:  []cli.Flag {
		utils.RevertToHeightFlag,
	},
}

func revertToHeight(ctx *cli.Context) error {
	revertHeight := ctx.GlobalInt(utils.GetFlagName(utils.RevertToHeightFlag))
	fmt.Println("revertHeight:", revertHeight)
	if revertHeight != 0 {
		err := blockrelayer.RevertToHeight(blockrelayer.DefStorage.GetMetaDB(), uint32(revertHeight))
		return err
	}
	return nil
}