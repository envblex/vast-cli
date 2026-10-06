# vast-cli (Go Implementation)

Vast.ai の公式 Python CLI (`vastai`) を解析し、Go 言語で依存ゼロ・超高速・省メモリなネイティブバイナリとして完全再実装した軽量 CLI ツールです。

## 特徴 & パフォーマンス比較

| 項目 | 公式 Python 版 (`vastai`) | Go 再実装版 (`vast`) | 改善効果 |
| :--- | :--- | :--- | :--- |
| **起動時間 (`--version`)** | 約 2.3 秒 | **0.003 秒 (3ms)** | **約 770 倍高速化** |
| **ヘルプ表示 (`--help`)** | 約 0.85 秒 | **0.015 秒 (15ms)** | **約 55 倍高速化** |
| **ディスク容量** | **189 MB** (`.venv`) | **6.4 MB** (単一バイナリ) | **約 30 分の 1 に軽量化** |
| **外部依存パッケージ** | 30個以上 (borb, pillow, cryptography 等) | **ゼロ** (標準ライブラリのみ) | 依存関係の競合・脆弱性リスク排除 |
| **実行環境** | Python 3.9+ 仮想環境必須 | 単一バイナリで即座に実行可能 | 環境構築・ポータビリティが大幅向上 |

---

## ディレクトリ構成

```
.
├── .gitignore
├── .venv/                 # 公式 Python 版の検証・解析用仮想環境
├── requirements.txt       # 公式 Python 版の依存関係スナップショット
├── Makefile               # ビルド・インストール自動化
├── go.mod
├── cmd/
│   └── vast/
│       └── main.go        # CLI エントリーポイント & サブコマンドディスパッチャ
└── pkg/
    ├── api/               # Vast.ai REST API クライアントロジック
    │   ├── offers.go      # GPU オファー検索
    │   ├── instances.go   # インスタンス CRUD / 制御 / ログ / SSH接続生成
    │   ├── user.go        # アカウント情報・残高
    │   ├── sshkeys.go     # SSH 公開鍵登録・一覧・削除
    │   ├── envvars.go     # 環境変数シークレット管理
    │   └── templates_volumes.go # テンプレート / ボリューム検索
    ├── client/            # HTTP クライアント (リトライ、--explain, --curl, ポーリング)
    ├── config/            # APIキー解決 & 設定ディレクトリ管理
    ├── display/           # マルチブロック表形式整形 & --raw JSON 出力
    └── query/             # Vast クエリ構文パーサー (gpu_name, 演算子, 単位変換)
```

---

## インストール & ビルド

### 1. Go バイナリのビルドとインストール
```bash
# ビルド (bin/vast および bin/vastai を生成)
make build

# ~/.local/bin/ にインストール (my CLI ハブからも呼び出し可能)
make install
```

### 2. Python 仮想環境の管理 (参照・互換性確認用)
```bash
# venv 作成と公式 vastai のインストール
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

---

## 認証設定 (API Key)

API キーは以下の優先順位で自動解決されます：
1. CLI 引数: `--api-key <KEY>`
2. 環境変数: `VAST_API_KEY`
3. XDG 設定ファイル: `~/.config/vastai/vast_api_key`
4. レガシー設定ファイル: `~/.vast_api_key`

```bash
# API キーの登録 (1回のみ実行、~/.config/vastai/vast_api_key に保存)
vast set api-key <YOUR_API_KEY>

# または my コマンドハブ経由
my vast set api-key <YOUR_API_KEY>
```

---

## コマンド一覧 & 使用例

### GPU オファー検索 (`search offers`)
```bash
# デフォルト検索 (スコア順、上位5件)
vast search offers --limit 5

# クエリ構文による条件検索
vast search offers "gpu_name=RTX_5090 num_gpus=1 verified=true" --limit 3

# ソート順の指定 (dlperf_usd 降順)
vast search offers "gpu_ram>=24 reliability>0.98" -o "dlperf_usd-" --limit 5

# JSON 形式で出力 (--raw)
vast search offers "gpu_name=RTX_4090" --limit 1 --raw
```

### インスタンス管理 (`instances`)
```bash
# インスタンス一覧表示
vast show instances

# フィルタリング
vast show instances --status running
vast show instances --gpu-name "RTX 4090"

# 単一インスタンスの詳細確認
vast show instance <INSTANCE_ID>

# インスタンスの作成 (オファーID指定)
vast create instance <OFFER_ID> --image pytorch/pytorch:latest --disk 20 --ssh --direct

# 起動 / 停止 / 再起動
vast start instance <INSTANCE_ID>
vast stop instance <INSTANCE_ID>
vast reboot instance <INSTANCE_ID>

# インスタンスの破棄 (-y で確認プロンプトをスキップ)
vast destroy instance <INSTANCE_ID> -y
```

### SSH 接続 URL の取得
```bash
# ssh:// 接続文字列を取得
vast ssh-url <INSTANCE_ID>
# 出力例: ssh://root@123.45.67.89:2222

# scp:// 接続文字列を取得
vast scp-url <INSTANCE_ID>
```

### ログ取得 & リモート実行
```bash
# 直近 100 行のログを取得
vast logs <INSTANCE_ID> --tail 100

# インスタンス上でコマンド実行
vast execute <INSTANCE_ID> "nvidia-smi"
```

### アカウント & SSH 鍵管理
```bash
# ユーザー情報・クレジット残高確認
vast show user

# SSH 公開鍵の一覧
vast show ssh-keys

# SSH 公開鍵の登録 (ファイル指定または文字列)
vast create ssh-key ~/.ssh/id_ed25519.pub
```

### グローバルフラグ
- `--raw`: すべてのコマンドで機械可読な JSON を出力
- `--explain`: 実行される HTTP リクエスト (URL, Headers, Body) を詳細表示
- `--curl`: 実行と同等の `curl` コマンドを出力して終了
- `--url <URL>`: Vast.ai サーバーの REST API URL を上書き
- `--retry <N>`: 一時的エラー時のリトライ回数を指定 (デフォルト: 3)
- `--version`: バージョン表示
