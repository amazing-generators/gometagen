package git

import _ "embed"

// // // // // // // // // //

var (
	//go:embed hook_commit_msg.sh
	CommitMsgHookArr []byte

	//go:embed hook_pre_push.sh
	PrePushHookArr []byte

	//go:embed hook_noop.sh
	NoopHookArr []byte
)
