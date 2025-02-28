package flags

import (
	"gabe565.com/utils/must"
	"github.com/clevyr/kubedb/internal/consts"
	"github.com/clevyr/kubedb/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RemoteGzip(cmd *cobra.Command) {
	cmd.Flags().Bool(consts.FlagRemoteGzip, true, "Compress data over the wire. Results in lower bandwidth usage, but higher database load. May improve speed on slow connections.")
	must.Must(cmd.RegisterFlagCompletionFunc(consts.FlagRemoteGzip, util.BoolCompletion))
}

func BindRemoteGzip(cmd *cobra.Command) {
	must.Must(viper.BindPFlag(consts.KeyRemoteGzip, cmd.Flags().Lookup(consts.FlagRemoteGzip)))
}
