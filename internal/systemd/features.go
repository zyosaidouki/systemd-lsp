package systemd

import (
	"regexp"
	"strconv"
	"strings"
)

// These refinements supplement the gperf parser types. Sources are the tagged
// upstream manuals: v261/man/systemd.{exec,service,unit,resource-control}.xml
// and v262/man/systemd.{exec,service,slice}.xml. v263 is not released yet; its
// refinements follow man/systemd.exec.xml and src/shared/io-uring-filter.c at
// commit 9571d664d89c of the development branch. Gperf remains authoritative
// for which directives exist in a catalog; refinements never add directives.
type featureSpec struct {
	sections   []string
	since      int
	kind       string
	values     []string
	valueSince map[string]int
	doc        string
	ja         string
	example    string
	syntax     string
	snippet    string
	man        string
	multiple   bool
}

var execSections = []string{"Service", "Socket", "Mount", "Swap"}
var resourceSections = []string{"Service", "Socket", "Mount", "Swap", "Slice", "Scope"}

func withBooleans(values ...string) []string {
	return append(append([]string{}, boolValues...), values...)
}

// io_uring_op_table, in opcode order: IORING_OP_ constants in lower case.
var ioUringOperations = []string{
	"nop", "readv", "writev", "fsync", "read_fixed", "write_fixed", "poll_add", "poll_remove",
	"sync_file_range", "sendmsg", "recvmsg", "timeout", "timeout_remove", "accept", "async_cancel",
	"link_timeout", "connect", "fallocate", "openat", "close", "files_update", "statx", "read", "write",
	"fadvise", "madvise", "send", "recv", "openat2", "epoll_ctl", "splice", "provide_buffers",
	"remove_buffers", "tee", "shutdown", "renameat", "unlinkat", "mkdirat", "symlinkat", "linkat",
	"msg_ring", "fsetxattr", "setxattr", "fgetxattr", "getxattr", "socket", "uring_cmd", "send_zc",
	"sendmsg_zc", "read_multishot", "waitid", "futex_wait", "futex_wake", "futex_waitv",
	"fixed_fd_install", "ftruncate", "bind", "listen", "recv_zc", "epoll_wait", "readv_fixed",
	"writev_fixed", "pipe", "nop128", "uring_cmd128",
}

var featureSpecs = map[string]featureSpec{
	"PrivateUsers": {
		sections: execSections, since: 232, kind: "enum",
		values:     withBooleans("self", "identity", "managed", "full"),
		valueSince: map[string]int{"self": 257, "identity": 257, "managed": 260, "full": 258},
		doc:        "Creates a user namespace. self maps root and the service user; identity maps the first 65536 IDs; managed allocates an ID range; full maps the entire UID/GID range (v258+).",
		ja:         "ユーザー名前空間を作成します。self は root とサービスのユーザー、identity は先頭65536個のID、managed は動的に割り当てたID範囲、full はUID/GID全域をマッピングします（full は v258 以降）。",
		example:    "PrivateUsers=yes", man: "systemd.exec(5)",
	},
	"CPUSetPartition": {
		sections: resourceSections, since: 261, kind: "enum", values: []string{"member", "root", "isolated"},
		doc:     "Selects a cpuset partition: member for normal membership, root for a partition root, or isolated for CPU isolation. Also requires AllowedCPUs=; the kernel default is member.",
		ja:      "CPU集合の区分を指定します。member は通常の所属、root は区分のルート、isolated はCPUの分離です。AllowedCPUs= の指定も必要です。カーネルの既定値は member です。",
		example: "AllowedCPUs=2-3\nCPUSetPartition=isolated", man: "systemd.resource-control(5)",
	},
	"CPUPressureWatch": {
		sections: resourceSections, since: 261, kind: "enum", values: withBooleans("auto", "skip"),
		doc:     "Configures CPU pressure notifications. auto enables monitoring when CPU resource controls are configured; skip leaves the notification environment untouched. The service must implement pressure handling.",
		ja:      "CPUの待ち時間に関する負荷通知を設定します。auto はCPUリソース制御がある場合に有効化し、skip は通知用環境変数を設定しません。通知を処理するにはサービス側の対応が必要です。",
		example: "CPUPressureWatch=yes\nCPUPressureThresholdSec=200ms", man: "systemd.resource-control(5)",
	},
	"IOPressureWatch": {
		sections: resourceSections, since: 261, kind: "enum", values: withBooleans("auto", "skip"),
		doc:     "Configures I/O pressure notifications. auto enables monitoring when I/O accounting is enabled; skip leaves the notification environment untouched. The service must implement pressure handling.",
		ja:      "I/Oの待ち時間に関する負荷通知を設定します。auto はI/O使用量の集計が有効な場合に有効化し、skip は通知用環境変数を設定しません。通知を処理するにはサービス側の対応が必要です。",
		example: "IOPressureWatch=yes\nIOPressureThresholdSec=200ms", man: "systemd.resource-control(5)",
	},
	"CPUPressureThresholdSec": {
		sections: resourceSections, since: 261, kind: "duration",
		doc:     "CPU stall time that triggers a notification within a two-second window. Used with CPUPressureWatch=. Accepts time units such as ms and us; the manager default is 200ms.",
		ja:      "2秒間の測定窓で通知を発生させるCPU待ち時間の閾値です。CPUPressureWatch= と併用します。ms や us などの時間単位を使え、マネージャーの既定値は 200ms です。",
		example: "CPUPressureThresholdSec=200ms", man: "systemd.resource-control(5)",
	},
	"IOPressureThresholdSec": {
		sections: resourceSections, since: 261, kind: "duration",
		doc:     "I/O stall time that triggers a notification within a two-second window. Used with IOPressureWatch=. Accepts time units such as ms and us; the manager default is 200ms.",
		ja:      "2秒間の測定窓で通知を発生させるI/O待ち時間の閾値です。IOPressureWatch= と併用します。ms や us などの時間単位を使え、マネージャーの既定値は 200ms です。",
		example: "IOPressureThresholdSec=200ms", man: "systemd.resource-control(5)",
	},
	"FileDescriptorStorePreserve": {
		sections: []string{"Service"}, since: 254, kind: "enum",
		// exec_preserve_mode_from_string() takes every boolean spelling.
		values:     []string{"no", "yes", "restart", "on-success", "true", "false", "on", "off", "1", "0"},
		valueSince: map[string]int{"on-success": 261},
		doc:        "Controls file descriptor store retention. no releases it at stop, restart retains it during restart, and yes retains it until explicit cleanup. on-success (v261+) discards it only on permanent failure, after automatic restarts are exhausted.",
		ja:         "ファイルディスクリプタストアの保持条件です。no は停止時に解放、restart は再起動中に保持、yes は明示的な削除まで保持します。on-success（v261以降）は自動再起動を使い切った最終的な失敗時に解放します。",
		example:    "FileDescriptorStoreMax=16\nFileDescriptorStorePreserve=restart", man: "systemd.service(5)",
	},
	"RefreshOnReload": {
		sections: []string{"Service"}, since: 260, kind: "resource-list", values: withBooleans("extensions", "credentials"), multiple: true,
		doc:     "Refreshes extension images/directories and/or credentials on service reload. Accepts a boolean or a space-separated resource list; a leading ~ removes resources. Empty assignment restores the default, extensions.",
		ja:      "再読み込み時に拡張イメージ・ディレクトリや資格情報を更新します。真偽値または extensions / credentials の空白区切りリストを指定します。先頭の ~ は対象の除外、空の代入は既定値 extensions へのリセットです。",
		example: "RefreshOnReload=extensions credentials", syntax: "RefreshOnReload=yes|no|[~]extensions credentials", man: "systemd.service(5)",
	},
	"ExecReloadPost": {
		sections: []string{"Service"}, since: 259, kind: "command", multiple: true,
		doc:     "Commands executed after the service configuration reload completes.",
		ja:      "サービスの設定の再読み込みが完了した後に実行するコマンドです。",
		example: "ExecReloadPost=/usr/bin/logger configuration-reloaded", man: "systemd.service(5)",
	},
	"ConditionFraction": {
		sections: []string{"Unit"}, since: 261, kind: "fraction", multiple: true,
		doc:     "Selects a stable fraction of machines using the machine ID. Specify an optional rollout tag and a percentage from 0% to 100%, with up to two decimal places. | makes a trigger condition and ! negates the match.",
		ja:      "マシンIDから決まる一定割合のマシンで実行する条件です。任意の配備タグと、0%〜100%（小数点以下2桁まで）を指定します。| はトリガー条件、! は条件の反転です。",
		example: "ConditionFraction=rollout 10%", syntax: "ConditionFraction=[|][!][tag ]percentage%", snippet: "ConditionFraction=${1:rollout} ${2:10}%$0", man: "systemd.unit(5)",
	},
	"ConditionMachineTag": {
		sections: []string{"Unit"}, since: 261, kind: "condition", multiple: true,
		doc:     "Matches a machine-info TAGS entry using shell-style globbing (*, ?, []). A leading ! negates the match; | makes a trigger condition.",
		ja:      "machine-info の TAGS にあるタグを照合します。*、?、[] のパターンを使えます。先頭の ! は条件の反転、| はトリガー条件です。",
		example: "ConditionMachineTag=production-*", syntax: "ConditionMachineTag=[|][!]pattern", snippet: "ConditionMachineTag=${1:production-*}$0", man: "systemd.unit(5)",
	},
	"RestartRandomizedDelaySec": {
		sections: []string{"Service"}, since: 262, kind: "duration",
		doc:     "Adds a uniformly random delay to RestartSec= and any exponential backoff on automatic restart, spreading simultaneous restarts.",
		ja:      "自動再起動時に RestartSec= や指数バックオフへランダムな遅延を加え、一斉再起動を分散します。",
		example: "Restart=on-failure\nRestartSec=5s\nRestartRandomizedDelaySec=30s", man: "systemd.service(5)",
	},
	"ActivatingConcurrencyMax": {
		sections: []string{"Slice"}, since: 262, kind: "concurrency", values: []string{"infinity"},
		doc:     "Limits simultaneous activations in this slice hierarchy. Excess starts wait in the queue; infinity disables the limit. 0 blocks all starts until raised. Configure service startup timeouts to avoid occupied slots being stuck.",
		ja:      "スライス階層内で同時に起動処理できるユニット数を制限します。超過分は待機し、infinity は無制限です。0 は値を上げるまで全起動を保留します。起動枠が塞がれ続けないようサービス側に起動タイムアウトを設定します。",
		example: "ActivatingConcurrencyMax=4", syntax: "ActivatingConcurrencyMax=<unsigned integer>|infinity", man: "systemd.slice(5)",
	},
	"NUMAPolicy": {
		sections: execSections, since: 243, kind: "enum", values: []string{"default", "preferred", "bind", "interleave", "local", "preferred-many", "weighted-interleave"},
		valueSince: map[string]int{"preferred-many": 262, "weighted-interleave": 262},
		doc:        "Sets the NUMA memory policy; NUMAMask= selects nodes. preferred-many requires Linux 5.15+; weighted-interleave requires Linux 6.9+ and kernel-configured weights. Both new policies require systemd v262.",
		ja:         "NUMAメモリ配置方針です。対象ノードは NUMAMask= で指定します。preferred-many は Linux 5.15 以降、weighted-interleave は Linux 6.9 以降とカーネル側の重み設定が必要です。この2値は systemd v262 以降で使えます。",
		example:    "NUMAPolicy=bind\nNUMAMask=0", man: "systemd.exec(5)",
	},
	// Booleans that later gained further values. Treating them as plain
	// booleans would reject the newer spellings.
	"PrivateTmp": {
		sections: execSections, kind: "enum", values: withBooleans("disconnected"),
		valueSince: map[string]int{"disconnected": 257},
	},
	"ProtectHome": {
		sections: execSections, kind: "enum", values: withBooleans("read-only", "tmpfs"),
	},
	"ProtectSystem": {
		sections: execSections, kind: "enum", values: withBooleans("full", "strict"),
	},
	"ProtectControlGroups": {
		sections: execSections, kind: "enum", values: withBooleans("private", "strict"),
		valueSince: map[string]int{"private": 257, "strict": 257},
	},
	"ProtectHostname": {
		sections: execSections, kind: "enum", values: withBooleans("private"),
		valueSince: map[string]int{"private": 258},
	},
	"BindIPv6Only": {sections: []string{"Socket"}, kind: "enum"},
	"SecureBits": {
		sections: execSections, kind: "enum-list", multiple: true,
		values:     []string{"keep-caps", "keep-caps-locked", "no-setuid-fixup", "no-setuid-fixup-locked", "noroot", "noroot-locked", "no-cap-ambient-raise", "no-cap-ambient-raise-locked", "exec-restrict-file", "exec-restrict-file-locked", "exec-deny-interactive", "exec-deny-interactive-locked"},
		valueSince: map[string]int{"no-cap-ambient-raise": 262, "no-cap-ambient-raise-locked": 262, "exec-restrict-file": 262, "exec-restrict-file-locked": 262, "exec-deny-interactive": 262, "exec-deny-interactive-locked": 262},
		doc:        "Space-separated securebits flags. Repeated assignments combine flags; an empty assignment resets them. no-cap-ambient-raise, exec-restrict-file, exec-deny-interactive and their locked variants require systemd v262.",
		ja:         "securebits フラグを空白区切りで指定します。複数回の代入はフラグを結合し、空の代入でリセットします。no-cap-ambient-raise、exec-restrict-file、exec-deny-interactive と各 locked 版は systemd v262 以降で使えます。",
		example:    "SecureBits=keep-caps no-setuid-fixup", syntax: "SecureBits=<flag> ...", man: "systemd.exec(5)",
	},
	"IOUringFilter": {
		sections: execSections, since: 263, kind: "io-uring-filter", values: ioUringOperations, multiple: true,
		doc:     "Restricts the io_uring operations the unit's processes may submit. Takes operation names separated by spaces or commas; a leading ~ makes it a deny list, otherwise only the listed operations are allowed. Repeated assignments are merged, an assignment of the other type removes operations, and an empty assignment resets the list. Denied requests fail with EACCES. Requires Linux 7.0+; on older kernels io_uring_setup() fails with ENOSYS instead. Preview support based on the systemd development branch for v263.",
		ja:      "ユニットのプロセスが発行できる io_uring 操作を制限します。操作名を空白またはカンマ区切りで指定します。先頭に ~ を付けると拒否リスト、付けなければ列挙した操作だけを許可するリストです。複数回の代入は結合され、逆の種類の代入は操作をリストから取り除き、空の代入でリセットします。拒否された要求は EACCES で失敗します。Linux 7.0 以降が必要で、古いカーネルでは代わりに io_uring_setup() が ENOSYS で失敗します。systemd v263 の開発版に基づく先行対応です。",
		example: "IOUringFilter=~uring_cmd uring_cmd128 send_zc sendmsg_zc recv_zc", syntax: "IOUringFilter=[~]operation ...", man: "systemd.exec(5)",
	},
}

var majorVersionRE = regexp.MustCompile(`^v?([0-9]+)(?:[.-].*)?$`)

func majorVersion(version string) int {
	matches := majorVersionRE.FindStringSubmatch(version)
	if len(matches) == 0 {
		return 0 // Unversioned and development snapshots have no numeric cutoff.
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}

// EnrichCatalogFeatures also handles old generated files that lack value and
// introduction metadata. Explicit values are retained, with known newer values
// excluded when the file targets an older release.
func EnrichCatalogFeatures(file CatalogFile) CatalogFile {
	version := majorVersion(file.Version)
	out := make([]CatalogDirective, 0, len(file.Directives))
	for _, d := range file.Directives {
		// Catalogs generated before these rules existed are corrected on load.
		d.Multiple = d.Multiple || inferMultiple(d.Parser, d.Name)
		if d.Parser == "config_parse_tristate" {
			d.Values = boolValues
		}
		if d.Parser == "config_parse_socket_bind" {
			d.Values = inferValues(d.Section, d.Parser, d.Name)
		}
		name := d.Name
		if name == "AssertFraction" || name == "AssertMachineTag" {
			name = "Condition" + strings.TrimPrefix(name, "Assert")
		}
		spec, ok := featureSpecs[name]
		if ok && contains(spec.sections, d.Section) {
			if d.Since == 0 {
				d.Since = spec.since
			}
			d.ValueKind = spec.kind
			// A generated snapshot takes the current refinement, even if it
			// was generated before the refinement last changed. The values
			// of a hand-written overlay are kept.
			if len(spec.values) > 0 && (version > 0 || len(d.Values) == 0) {
				d.Values = spec.values
			}
			values := make([]string, 0, len(d.Values))
			for _, value := range d.Values {
				if version == 0 || spec.valueSince[value] <= version {
					values = append(values, value)
				}
			}
			d.Values = values
			if d.Doc == "" {
				d.Doc = spec.doc
			}
			if d.DocJapanese == "" {
				d.DocJapanese = spec.ja
			}
			if d.Example == "" {
				d.Example = strings.ReplaceAll(spec.example, name, d.Name)
			}
			if d.Syntax == "" {
				d.Syntax = strings.ReplaceAll(spec.syntax, name, d.Name)
			}
			if d.Snippet == "" {
				d.Snippet = strings.ReplaceAll(spec.snippet, name, d.Name)
			}
			if d.ManPage == "" {
				d.ManPage = spec.man
			}
			d.Multiple = d.Multiple || spec.multiple
		}
		if version > 0 && d.Since > version {
			continue
		}
		out = append(out, d)
	}
	file.Directives = out
	return file
}
