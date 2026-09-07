# systemd-lsp

[English](README.md) | 日本語

systemd のユニットファイル向けの小さな Language Server Protocol（LSP）実装です。

標準入出力で通信し、Neovim、Vim、gVim、その他の LSP クライアントから利用できます。

## 機能

- systemd ユニットファイルでよくある間違いの診断
  - セクションより前に記述されたキー
  - 不正なセクションヘッダー
  - 未知のセクション
  - 既知のセクション内にある未知のディレクティブ
  - 複数回指定できないディレクティブの重複
- 主なセクション・ディレクティブの補完（構文と使用例を含む）
- systemd のパーサーソースから生成したディレクティブカタログの組み込み
- 特定の systemd バージョンの man XML ドキュメントを含む外部カタログへの対応
- 主な選択肢型ディレクティブの値補完
- 既知のディレクティブのホバードキュメント
- セクションとディレクティブのドキュメントシンボル
- 空の単独 `.service` ファイルへのテンプレート挿入。ドロップインや他のユニット種別には自動挿入しません

## 動作要件

- Neovim 0.11 以降では、組み込みの LSP クライアントを使用します。
- Vim／gVim 8.0 以降では、LSP クライアントプラグインが必要です。同梱の連携機能は
  `prabirshrestha/vim-lsp` を使用し、Vim の `+job`、`+channel`、タイマー、ラムダ式、JSON サポートを必要とします。
- その他のエディタでは、標準入出力でサーバーを起動できる LSP クライアントが必要です。
- systemd ユニットファイルは通常 Linux で使われるため、主な対象は Linux です。
  サーバー自体は Go のみで実装され、実行時に `systemd`、`systemctl`、`systemd-analyze` を呼び出しません。
- ローカルへの systemd のインストール、systemd ソースツリー、man ページ、C コンパイラ、
  systemd 開発用ヘッダーは不要です。標準のディレクティブカタログは実行ファイルに組み込まれています。

## インストールと更新

以下の [Neovim](#neovim) または [Vim／gVim](#vim-and-gvim) の手順でエディタプラグインをインストールしてください。
初回読み込み時に、プラグインが GitHub Releases の最新版を非同期でダウンロードし、
**プラグインのディレクトリ内にある `bin/systemd-lsp`** に保存して LSP クライアントを起動します。
実行ファイルの個別インストール、PATH 設定、Go ツールチェーン、`~/go` ディレクトリは不要です。
実行ファイルを `~/.local/bin` に配置することもありません。

ビルド済みリリースは Linux x86-64 と macOS Apple Silicon（arm64）に対応しています。
インストールには `sh`、`curl`、`tar`、`sha256sum` または `shasum`、GitHub へのネットワーク接続、
プラグインディレクトリへの書き込み権限が必要です。GitHub へのサインインは不要です。
実行ファイルを置き換える前にチェックサムを検証します。ダウンロードや検証に失敗した場合は、既存の実行ファイルを保持します。

Vim／Neovim から更新するには、次のコマンドを実行します。

```vim
:SystemdLspUpdate
```

初回インストールに失敗した場合も、このコマンドで再試行できます。
インストール・更新に成功すると、プラグインが LSP クライアントを起動または再起動します。
実行ファイルのパスやシェルコマンドを入力する必要はありません。
通常のエディタ起動時はインストール済みの実行ファイルを再利用し、更新確認は行いません。
プラグインを削除すると、その管理下の実行ファイルも削除されます。
プラグインマネージャーが Git の管理対象外である `bin/` を消去した場合は、次回読み込み時に再ダウンロードします。

現在の `main` コミットでテストとビルドが成功すると、CI が Linux・Mac 用のアーカイブと `SHA256SUMS` を
[GitHub Releases](https://github.com/zyosaidouki/systemd-lsp/releases/latest) に公開します。
各リリースには `build-<実行番号>-<試行番号>` 形式のタグが付きます。プルリクエストではリリースを公開しません。

### 既存の設定からの移行

エディタ設定から、独自の `SystemdLspUpdate` コマンド定義、`go install` のビルドフック、
固定の実行ファイルパスを削除してください。Neovim では、以前の `lsp/systemd.lua` と
`vim.lsp.enable("systemd")` の呼び出しも削除します。
現在はプラグイン自身が `systemd-lsp` クライアントを登録・有効化します。
以下の最小限のプラグイン設定を使用してください。言語とカタログの設定は、記載のグローバル変数で引き継げます。

プラグイン管理のクライアントが動作することを確認したら、以前の単独の実行ファイルは削除できます。
プラグインが他のインストールを削除・変更したり、Go のグローバル設定を変更したりすることはありません。

## セットアップの確認

設定は各マシンで初回だけ必要です。プラグインをインストールしたら `.service` ファイルを開き、
初回ダウンロードが完了するまで待ちます。次のコマンドでファイルタイプを確認してください。

```vim
:set filetype?
```

結果が `systemd` になれば、ファイルタイプの検出は成功です。Neovim では次のコマンドでクライアントを確認できます。

```vim
:lua print(vim.lsp.get_clients({ name = "systemd-lsp" }))
```

Vim／gVim では `:LspStatus` を使用します。
新しい空の単独 `.service` ファイルを開くと、テンプレートが挿入されます。
既存の内容があるファイルやドロップインには挿入しません。
インストールに失敗した場合は `:messages` を確認し、`:SystemdLspUpdate` で再試行してください。

### 対応する拡張子

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

`.network`、`.netdev`、`.link`、`.nspawn` など、ユニットファイルではない systemd 関連形式は対象外です。
`.device` ユニットも、現在のカタログとエディタのファイルタイプ検出ルールには含まれていません。

## Neovim

Neovim 0.11 以降が必要です。lazy.nvim を使用する場合は、次のプラグイン設定を追加します。
`lua/plugins` を読み込む構成では、`~/.config/nvim/lua/plugins/systemd.lua` として保存してください。

```lua
return {
  {
    "zyosaidouki/systemd-lsp",
    lazy = false,
    init = function()
      vim.g.systemd_lsp_locale = "ja" -- 任意。既定は英語
    end,
  },
}
```

`:Lazy sync` を実行して Neovim を再起動します。
`build` フック、独自の更新コマンド、`lsp/systemd.lua`、`vim.lsp.enable` の呼び出しは不要です。
プラグインがファイルタイプ検出と組み込み LSP クライアントを自動設定します。

プラグインマネージャーを使わない場合は、Neovim 標準のパッケージディレクトリにクローンします。

```sh
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.config/nvim/pack/lsp/start/systemd-lsp
```

Neovim を再起動してください。言語や任意の外部ディレクティブカタログを指定するには、
プラグイン読み込み前の `init.lua`、または lazy.nvim の `init` コールバックで設定します。

```lua
vim.g.systemd_lsp_locale = "ja" -- 英語にする場合は "en"
vim.g.systemd_lsp_catalog_path = "/path/to/catalog.json" -- 任意
```

<a id="vim-and-gvim"></a>

## Vim／gVim

`+job`、`+channel`、タイマー、ラムダ式、JSON サポートを備えた Vim／gVim 8.0 以降と、
LSP クライアントプラグイン `prabirshrestha/vim-lsp` が必要です。
vim-plug を使用する場合は、次の設定を `~/.vimrc` に追加します。

```vim
filetype plugin on
let g:systemd_lsp_locale = 'ja' " 任意。既定は英語
call plug#begin()
Plug 'prabirshrestha/vim-lsp'
Plug 'zyosaidouki/systemd-lsp'
call plug#end()
```

`:PlugInstall` を実行して Vim／gVim を再起動してください。
プラグインが自身の管理下にある実行ファイルを使うため、`go install` フックや
`g:systemd_lsp_command` の設定は追加しないでください。

プラグインマネージャーを使わない場合は、両方を標準パッケージとしてインストールします。

```sh
mkdir -p ~/.vim/pack/lsp/start
git clone https://github.com/prabirshrestha/vim-lsp \
  ~/.vim/pack/lsp/start/vim-lsp
git clone https://github.com/zyosaidouki/systemd-lsp \
  ~/.vim/pack/lsp/start/systemd-lsp
```

`~/.vimrc` に `filetype plugin on` を追加して再起動します。
どちらのインストール方法でも `:SystemdLspUpdate` を使用できます。
補完は `Ctrl-X Ctrl-O`、ホバーは `:LspHover`、シンボル一覧は `:LspDocumentSymbol` で利用できます。

任意の外部カタログを指定する場合は、プラグイン読み込み前に次を設定してください。

```vim
let g:systemd_lsp_catalog_path = '/path/to/catalog.json'
```

## 開発

ソースからのビルドは任意で、Go 1.26 以降が必要です。
プラグイン管理の実行ファイルをローカルでビルドするには、次を実行します。

```sh
mkdir -p bin
go build -o bin/systemd-lsp ./cmd/systemd-lsp
```

その他の LSP クライアントでは、手動ダウンロードしたリリースやソースからビルドした実行ファイルを使用できます。
単独の実行ファイルでは引き続き `systemd-lsp update` を利用できます。
エディタプラグインの利用者は `:SystemdLspUpdate` を使用してください。

GitHub Actions は `main` への push と同一リポジトリからのプルリクエストに対して、
Linux x64 の self-hosted runner でテスト、`go vet`、ビルドを実行します。
フォークからのプルリクエストは self-hosted runner ではスキップします。
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

チェッカーはカタログの統計を表示し、セクション・ディレクティブの重複、空の名前、
指定した最小件数を下回るディレクティブ数など、明らかな問題があれば失敗として終了します。
