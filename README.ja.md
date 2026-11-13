# systemd-lsp

[English](README.md) | 日本語

systemd ユニットファイルの補完・診断・ホバー表示を提供する Vim／Neovim 用プラグインです。

[Linux x86-64 のセットアップ](#linux) | [macOS Apple Silicon のセットアップ](#macos)

<a id="linux"></a>

## Linux x86-64 のセットアップ

### 1. 使用するエディタとコマンドを確認する

- Neovim：0.11 以降
- Vim／gVim：8.0 以降（`+job`、`+channel`、タイマー、ラムダ式、JSON 対応）

ターミナルで次を実行します。

```sh
uname -m
command -v git curl tar sh sha256sum
```

`uname -m` が `x86_64` で、各コマンドのパスが表示されることを確認してください。
不足しているコマンドは、お使いの OS のパッケージ管理方法でインストールします。初回起動時は GitHub への接続が必要です。

### 2. エディタにプラグインを追加する

使用するエディタの手順を実行してください。すでに lazy.nvim または vim-plug を使っている場合は、
[プラグインマネージャーでの設定](#plugin-managers)を使い、この手順のクローン操作は省略します。

#### Neovim

ターミナルで実行します。

```sh
mkdir -p ~/.config/nvim/pack/lsp/start
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

`~/.config/nvim/init.lua` に次を追記します（ファイルがなければ作成します）。

```lua
vim.g.systemd_lsp_locale = "ja"
```

#### Vim／gVim

ターミナルで本体と LSP クライアントをインストールします。

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

`~/.vimrc` に次を追記します（ファイルがなければ作成します）。

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

### 3. 起動して動作を確認する

エディタを再起動し、エディタ内で次を実行します。

```vim
:edit example.service
```

初回は実行ファイルを自動ダウンロードします。完了したら次を実行してください。

```vim
:set filetype?
```

`filetype=systemd` と表示されることを確認します。Neovim では次で LSP 接続を確認できます。

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

クライアントが表示されれば接続できています。Vim／gVim では `:LspStatus` を実行し、
`systemd-lsp` が `running` であることを確認します。

<a id="macos"></a>

## macOS Apple Silicon のセットアップ

### 1. 使用するエディタとコマンドを確認する

- Neovim：0.11 以降
- Vim／gVim：8.0 以降（`+job`、`+channel`、タイマー、ラムダ式、JSON 対応）

ターミナルで次を実行します。

```sh
uname -m
command -v git curl tar sh shasum
```

`uname -m` が `arm64` で、各コマンドのパスが表示されることを確認してください。
不足しているコマンドは、お使いの OS のパッケージ管理方法でインストールします。初回起動時は GitHub への接続が必要です。

### 2. エディタにプラグインを追加する

使用するエディタの手順を実行してください。すでに lazy.nvim または vim-plug を使っている場合は、
[プラグインマネージャーでの設定](#plugin-managers)を使い、この手順のクローン操作は省略します。

#### Neovim

ターミナルで実行します。

```sh
mkdir -p ~/.config/nvim/pack/lsp/start
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

`~/.config/nvim/init.lua` に次を追記します（ファイルがなければ作成します）。

```lua
vim.g.systemd_lsp_locale = "ja"
```

#### Vim／gVim

ターミナルで本体と LSP クライアントをインストールします。

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

`~/.vimrc` に次を追記します（ファイルがなければ作成します）。

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

### 3. 起動して動作を確認する

エディタを再起動し、エディタ内で次を実行します。

```vim
:edit example.service
```

初回は実行ファイルを自動ダウンロードします。完了したら次を実行してください。

```vim
:set filetype?
```

`filetype=systemd` と表示されることを確認します。Neovim では次で LSP 接続を確認できます。

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

クライアントが表示されれば接続できています。Vim／gVim では `:LspStatus` を実行し、
`systemd-lsp` が `running` であることを確認します。

<a id="plugin-managers"></a>

## プラグインマネージャーを使う場合

Linux・macOS 共通です。すでに導入済みのマネージャーに、次の設定を追加してください。

### lazy.nvim

`lua/plugins` を読み込む構成では、`~/.config/nvim/lua/plugins/systemd.lua` に保存します。
それ以外は、既存のプラグイン一覧に内側のプラグイン定義を追加してください。

```lua
return {
  {
    "zyosaidouki/systemd-lsp",
    lazy = false,
    init = function()
      vim.g.systemd_lsp_locale = "ja"
    end,
  },
}
```

`:Lazy sync` を実行したら、上記の「3. 起動して動作を確認する」に進みます。

### vim-plug

`~/.vimrc` の既存の `plug#begin()` と `plug#end()` の間に、次の2行を追加します。

```vim
Plug 'prabirshrestha/vim-lsp'
Plug 'zyosaidouki/systemd-lsp'
```

同じファイルに次の設定を追記します。

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja'
```

`:PlugInstall` を実行したら、上記の「3. 起動して動作を確認する」に進みます。

## 更新

エディタ内で実行します。

```vim
:SystemdLspUpdate
```

最新版を確認し、必要な場合にバイナリを更新して LSP を再起動します。
実行ファイルはプラグイン内の `bin/systemd-lsp` に保存されます。
更新時は `systemd-lsp updated: build-41-1 -> build-42-1` のように更新前後のバージョンを表示します。
最新版なら `systemd-lsp already: build-42-1` と表示し、再インストールや LSP の再起動は行いません。
単独の `systemd-lsp update` も同じ形式で表示します。

`bin/systemd-lsp --version` でインストール済みのバージョンを確認できます。
ローカルビルドは `dev`、バージョン情報がない旧バイナリからの更新では更新前を `unknown` と表示します。
初回インストールの更新前は `not-installed` です。

## うまく動かない場合

- ダウンロードに失敗する：`:messages` で原因を確認し、`:SystemdLspUpdate` で再試行します。
- LSP が二重に起動する：以前の `lsp/systemd.lua`、`vim.lsp.enable("systemd")`、独自の LSP 起動設定を削除します。
- 以前のインストール方法から移行する：`go install` フック、固定のバイナリパス、独自の `SystemdLspUpdate` 定義を削除し、上記の設定に置き換えます。

## 言語と外部カタログ

上記の設定は日本語表示です。英語にする場合は `ja` を `en` に変更してください。
`ja` の場合、日本語の説明がない設定には英語の説明を表示します
（man ページの説明を含めて生成した外部カタログを使う場合など）。
外部カタログを使う場合は、プラグイン読み込み前に次を設定します。

Neovim (`init.lua` または lazy.nvim の `init`):

```lua
vim.g.systemd_lsp_catalog_path = "/path/to/catalog.json"
```

Vim／gVim (`~/.vimrc`):

```vim
let g:systemd_lsp_catalog_path = '/path/to/catalog.json'
```

## 機能

- systemd ユニットファイルでよくある間違いの診断
  - セクションより前に記述されたキー
  - 不正なセクションヘッダー
  - 未知のセクション
  - 既知のセクション内にある未知のディレクティブ
  - 複数回指定できないディレクティブの重複
  - systemd が受け付けない値（未知の列挙値や大文字・小文字の誤り、
    時間・サイズ・アクセスモード・整数・リソース上限の書式誤り）
  - systemd が受け付けないユニット名（`After=network` のように種別の
    接尾辞がないものや、カンマ区切りの名前）
  - 行末コメント（systemd は値の一部として読みます）
  - `ExecStart=` などのコマンド設定に書かれたシェル構文（`> file`、`|`、`&&`、
    `$(...)` など。systemd はシェルを使わないため、そのまま引数として渡します）
  - `MemoryLimit=` など、systemd が無視する廃止済みの設定
  - systemd が読み込みを拒否するユニット（`ExecStart=` のないサービスや、
    必須の設定がないソケット・タイマー・パス・マウントユニット）
- 主なセクション・ディレクティブの補完（構文と使用例を含む）
- systemd のパーサーソースから生成したディレクティブカタログの組み込み
- 特定の systemd バージョンの man XML ドキュメントを含む外部カタログへの対応
- 主な選択肢型ディレクティブの値補完
- 既知のディレクティブのホバードキュメント
- セクションとディレクティブのドキュメントシンボル
- 空の単独 `.service` ファイルへのテンプレート挿入。ドロップインや他のユニット種別には自動挿入しません

### 新しい systemd 設定への対応

内蔵カタログは systemd の開発ブランチ（v263 開発版、コミット
[9571d664d89c](https://github.com/systemd/systemd/commit/9571d664d89c9576564f332933781687414eb370)）の
パーサーから生成しています。v262 までの項目はリリース済みで、v263 の項目は先行対応です。

- `PrivateUsers=full` を正しい値として扱い、対象バージョンに合う値を補完します。
- `CPUSetPartition=`、`CPUPressureWatch=`、`IOPressureWatch=`、`FileDescriptorStorePreserve=` に値補完と日本語・英語の説明を提供します。
- `RefreshOnReload=` と `SecureBits=` は空白区切りの次の値を補完します。`RefreshOnReload=~extensions credentials` の反転指定にも対応します。
- `ConditionFraction=`／`AssertFraction=`、`ConditionMachineTag=`／`AssertMachineTag=` に編集可能なスニペットを提供します。スニペット非対応クライアントでは通常の `キー=` を挿入します。
- 配備割合の範囲・小数桁数、列挙値、`ActivatingConcurrencyMax=` の数値を検証します。`CPUSetPartition=` に必要な `AllowedCPUs=` が同じ文書にない場合はヒントを表示します。
- `RestartRandomizedDelaySec=`、`ActivatingConcurrencyMax=`、`NUMAPolicy=` と `SecureBits=` の新しい値に v262 向けの説明を提供します。
- v263 で追加される `IOUringFilter=` に、io_uring 操作名の補完（空白・カンマ区切り、先頭の `~` による拒否リスト）と説明を提供します。systemd が無視する未知の操作名、大文字・小文字の誤り、先頭以外の `~` を警告します。
- `StateDirectory=` などのディレクトリ設定で `private/` 以下を指す項目を警告します。systemd はその項目を無視します。シンボリックリンク先の検査は v263 以降のカタログでのみ行います。

診断は開いている文書内の静的な検証です。他のユニットファイルやドロップインの統合、
実行先のカーネル機能の検出は行いません。

外部カタログに `version` がある場合、そのカタログのディレクティブ一覧で内蔵一覧を置き換えます。
既知の値の追加バージョンも考慮するため、例えば v260 向けでは
`FileDescriptorStorePreserve=on-success` を補完せず、入力時に警告します。
`version` のない外部カタログは追加・上書き用として扱います。

## 対応する拡張子

次の拡張子が `systemd` ファイルタイプとして検出されると、エディタが `systemd-lsp` を起動します。

| 拡張子 | ユニット固有のセクション |
| --- | --- |
| `.service` | `[Service]` |
| `.socket` | `[Socket]` |
| `.timer` | `[Timer]` |
| `.path` | `[Path]` |
| `.mount` | `[Mount]` |
| `.automount` | `[Automount]` |
| `.swap` | `[Swap]` |
| `.target` | ユニット固有のセクションなし |
| `.slice` | `[Slice]` |
| `.scope` | `[Scope]` |

親ディレクトリからこれらのユニット種別を判定できるドロップインファイルにも対応します。
例えば `example.service.d/override.conf` や `backup.timer.d/schedule.conf` で LSP が有効になります。
対応するドロップインディレクトリの外にある一般的な `.conf` ファイルでは有効になりません。
ドロップインでも補完と診断を利用できますが、テンプレートは挿入しません。
`ExecStart=` の欠落など、ユニット全体が必要な検査はドロップインでは行いません。

`systemctl edit` がエディタで開く一時ファイル（`.#example.serviceXXXXXX` や
`example.service.d/.#override.confXXXXXX`）は、保存先のユニットファイルまたは
ドロップインとして扱います。

`.network`、`.netdev`、`.link`、`.nspawn` など、ユニットファイルではない systemd 関連形式は対象外です。
`.device` ユニットも、現在のカタログとエディタのファイルタイプ検出ルールには含まれていません。

## 開発

ソースからのビルドは任意で、Go 1.27 以降が必要です。
プラグイン管理の実行ファイルをローカルでビルドするには、次を実行します。

```sh
mkdir -p bin
go build -o bin/systemd-lsp ./cmd/systemd-lsp
```

その他の LSP クライアントでは、手動ダウンロードしたリリースやソースからビルドした実行ファイルを使用できます。
単独の実行ファイルでは引き続き `systemd-lsp update` を利用できます。
エディタプラグインの利用者は `:SystemdLspUpdate` を使用してください。

JSON の処理には Go 1.27 の `encoding/json/v2` を使用します。
入力は有効な UTF-8 とし、オブジェクト内のキーの重複を認めず、フィールド名の大文字・小文字を区別します。
カタログファイルには JSON 値を 1 つだけ記述してください。
生成カタログの整形形式と、LSP 応答で必要な `null` の結果や空配列は維持します。

GitHub Actions は `main` への push と同一リポジトリからのプルリクエストに対して、
Linux x64 の self-hosted runner でテスト、`go vet`、ビルドを実行します。
フォークからのプルリクエストは self-hosted runner ではスキップします。
Dependabot のプルリクエストも同様です。ワークフローが使うアクションはコミットで
固定しており、新しいバージョンはプルリクエストのマージ後に初めて実行されます。
Actions タブから `CI` ワークフローを手動実行することもできます。
runner はオンラインで、`self-hosted`、`Linux`、`X64` ラベルを持つ必要があります。
Go は `go.mod` に指定されたバージョンをワークフローがインストールします。

```sh
go test ./...
go run ./cmd/systemd-lsp
```

## カタログの生成

systemd のユニットディレクティブは、リリースごとに追加・削除されます。
幅広い補完に対応するため、この言語サーバーは生成済みのパーサーカタログを標準で組み込んでいます。
特定の systemd リリースに合わせて補完・ホバードキュメントを固定するには、
対象の systemd ソースタグからカタログを生成してください。

カタログの生成には次を使用します。

- `src/core/load-fragment-gperf.gperf.in`：受け付けるセクション名・ディレクティブ名とパーサー関数
- `man/*.xml`：補完・ホバードキュメント
- 主なパーサー型から推定した値の種類と選択肢

チェックアウト済みの systemd ソースツリーからカタログを生成する例です。

```sh
git clone --depth 1 --branch v258 https://github.com/systemd/systemd /tmp/systemd-v258

go run ./cmd/systemd-lsp-generate-catalog \
  -version v258 \
  -man-dir /tmp/systemd-v258/man \
  /tmp/systemd-v258/src/core/load-fragment-gperf.gperf.in \
  > /tmp/systemd-v258-catalog.json

go run ./cmd/systemd-lsp-check-catalog \
  -min-directives 500 \
  /tmp/systemd-v258-catalog.json
```

man ページのドキュメントを含めず、単一ファイルから簡易的に生成する例です。

```sh
curl -fL \
  https://raw.githubusercontent.com/systemd/systemd/v258/src/core/load-fragment-gperf.gperf.in \
  -o /tmp/load-fragment-gperf.gperf.in

go run ./cmd/systemd-lsp-generate-catalog \
  -version v258 \
  /tmp/load-fragment-gperf.gperf.in \
  > /tmp/systemd-v258-catalog.json
```

生成したカタログを言語サーバーに読み込ませるには、次を実行します。

```sh
SYSTEMD_LSP_CATALOG=/tmp/systemd-v258-catalog.json go run ./cmd/systemd-lsp
```

ジェネレーターは systemd の gperf テンプレートにある `EXEC_CONTEXT_CONFIG_ITEMS`、
`CGROUP_CONTEXT_CONFIG_ITEMS`、`KILL_CONTEXT_CONFIG_ITEMS` のマクロ呼び出しを展開します。
その後、セクション、ディレクティブ、パーサー関数、推定した値の種類、構文、使用例、man ページ、
判明している選択肢、および複数回の代入が通常想定されるかどうかを含む JSON を出力します。

man XML の導入バージョンを `since` として取り込みます。対応する新機能には
日本語説明、スニペット、バージョン別の値候補も追加します。

チェッカーはカタログの統計を表示し、セクション・ディレクティブの重複、空の名前、
指定した最小件数を下回るディレクティブ数など、明らかな問題があれば失敗として終了します。
