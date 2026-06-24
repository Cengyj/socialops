package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsReservedEmail_DingTalkDomain(t *testing.T) {
	require.True(t, isReservedEmail("dingtalk-123@dingtalk-connect.invalid"))
	require.True(t, isReservedEmail("DINGTALK-456@DINGTALK-CONNECT.INVALID")) // case-insensitive
	require.False(t, isReservedEmail("real@dingtalk.com"))
}

func TestInferSignupSourceFromEmail(t *testing.T) {
	cases := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "dingtalk synthetic email",
			email: "dingtalk-user" + DingTalkConnectSyntheticEmailDomain,
			want:  "dingtalk",
		},
		{
			name:  "linuxdo synthetic email",
			email: "linuxdo-user" + LinuxDoConnectSyntheticEmailDomain,
			want:  "linuxdo",
		},
		{
			name:  "oidc synthetic email",
			email: "oidc-user" + OIDCConnectSyntheticEmailDomain,
			want:  "oidc",
		},
		{
			name:  "wechat synthetic email",
			email: "wechat-user" + WeChatConnectSyntheticEmailDomain,
			want:  "wechat",
		},
		{
			name:  "stored uppercase synthetic email",
			email: " LINUXDO-USER@LINUXDO-CONNECT.INVALID ",
			want:  "linuxdo",
		},
		{
			name:  "regular email",
			email: "user@example.com",
			want:  "email",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, inferSignupSourceFromEmail(tc.email))
		})
	}
}
