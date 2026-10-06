# vast-cli (Go Implementation)

[![Go Report Card](https://goreportcard.com/badge/github.com/envblex/vast-cli)](https://goreportcard.com/report/github.com/envblex/vast-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Vast.ai の公式 Python CLI (`vastai`) を解析し、Go 言語で依存ゼロ・超高速・省メモリな単一バイナリとして完全再実装した軽量 CLI ツールです。

> **初めて Vast.ai を利用される方へ**:  
> アカウント開設からクレジットのチャージ、GPU の検索、インスタンスの起動、SSH ログイン、そして最も重要な**課金の止め方**までをゼロから解説した [初心者向けスタートガイド (docs/getting-started.md)](docs/getting-started.md) を用意しています。

---

## パフォーマンス・リソース比較

公式 Python 版は多くの重いライブラリ（`pillow`, `borb`, `cryptography`, `aiohttp` など 30 個以上）に依存しており、単なるコマンド実行でも大きな遅延とディスク消費が発生します。Go 実装版は標準ライブラリのみで構成されており、圧倒的な速度と軽量性を誇ります。

| 項目 | 公式 Python 版 (`vastai`) | Go 再実装版 (`vast` / `vastai`) | 改善効果 |
| :--- | :--- | :--- | :--- |
| **起動時間 (`--version`)** | 約 2.31 秒 | **0.003 秒 (3ms)** | **約 770 倍高速化** |
| **ヘルプ表示 (`--help`)** | 約 0.85 秒 | **0.015 秒 (15ms)** | **約 55 倍高速化** |
| **ディスク容量** | **189 MB** (`.venv`) | **6.4 MB** (単一バイナリ) | **約 30 分の 1 に軽量化** |
| **外部依存パッケージ** | 30個以上 | **ゼロ** (Go 標準ライブラリのみ) | 依存衝突・脆弱性リスク完全排除 |
| **実行環境** | Python 3.9+ 仮想環境必須 | 単一静的バイナリ | 即座にポータブル実行可能 |

---

## クイックスタート (5分で GPU を借りて使う)

### 1. インストール
```bash
git clone https://github.com/envblex/vast-cli.git
cd vast-cli
make install
```
バイナリが `~/.local/bin/vast` および `~/.local/bin/vastai` に配置されます（ハブコマンド `my vast` からも実行可能）。

### 2. 認証設定 (初回のみ)
[Vast.ai Manage Keys](https://cloud.vast.ai/manage-keys/) で API キーをコピーし、設定します：
```bash
vast set api-key <YOUR_API_KEY>

# アカウント情報と残高の確認
vast show user
```

### 3. SSH 鍵の登録 (インスタンス作成前に必須)
```bash
vast create ssh-key ~/.ssh/id_ed25519.pub
```

### 4. GPU の検索
```bash
# 例: RTX 4090 × 1枚、信頼性95%以上
vast search offers "gpu_name=RTX_4090 num_gpus=1 reliability>0.95" -o "dlperf_usd-" --limit 3
```

### 5. インスタンスの作成 & 起動
```bash
# オファーIDを指定して作成 (Dockerイメージやディスクサイズを指定)
vast create instance <OFFER_ID> --image pytorch/pytorch:latest --disk 30 --ssh --direct

# 起動ステータスの確認 (actual_status が running になるまで待機)
vast show instances
```

### 6. SSH 接続
```bash
# SSH 接続 URL の取得
vast ssh-url <INSTANCE_ID>
# => ssh://root@192.0.2.1:34567

# そのまま接続
ssh root@192.0.2.1 -p 34567
```

### 7. インスタンスの破棄 (課金を完全停止)
```bash
# 作業完了後は必ず破棄してください (放置するとストレージ料金が課金され続けます)
vast destroy instance <INSTANCE_ID> -y
```

---

## コマンドリファレンス

### 検索コマンド (`search`)
```bash
# GPU オファー検索
vast search offers [query] [flags]
  --type on-demand|bid|reserved   契約形態の指定
  -o, --order FIELD[-]            ソート順 (例: 'dlperf_usd-', 'dph_total')
  --limit N                       取得件数の上限
  --storage GB                    算出用ストレージ容量 (デフォルト: 5.0)
  -n, --no-default                デフォルトフィルタ (verified, rentable) を無効化

# テンプレート / ボリューム / ベンチマーク検索
vast search templates [query]     公開・非公開テンプレート検索
vast search volumes [query]       ストレージボリューム検索
vast search benchmarks [query]    GPU ベンチマーク検索
```

### インスタンス管理 (`instances`)
```bash
vast show instances               インスタンス一覧表示 (--status, --gpu-name, --label)
vast show instance <ID>           単一インスタンスの詳細確認
vast create instance <OFFER_ID>   インスタンス作成 (--image, --disk, --ssh, --direct, --label 等)
vast destroy instance <ID...> -y  インスタンス破棄 (-y で確認スキップ)
vast start instance <ID...>       停止中インスタンスの起動
vast stop instance <ID...>        インスタンスの一時停止 (ディスク料金は継続)
vast reboot instance <ID>         インスタンスの再起動
vast recycle instance <ID>        インスタンスの再作成
vast label instance <ID> --label  インスタンスへのラベル付与
vast prepay instance <ID> <AMT>   リザーブドインスタンスへの事前入金
```

### SSH & 接続 (`ssh`)
```bash
vast ssh-url <ID>                 ssh:// 接続文字列の取得
vast scp-url <ID>                 scp:// 接続文字列の取得
vast attach ssh <ID> <KEY>        インスタンスへの SSH 公開鍵の追加アタッチ
vast detach ssh <ID> <KEY_ID>     インスタンスからの SSH 鍵デタッチ
vast show ssh-keys                アカウント登録済み SSH 公開鍵一覧
vast create ssh-key [KEY/PATH]    SSH 公開鍵の登録 (ファイルパスまたは文字列)
vast delete ssh-key <ID>          SSH 公開鍵の削除
```

### ログ & リモート実行 (`logs` / `execute`)
```bash
vast logs <ID> [--tail N] [--filter STR]  コンテナログ取得 (非同期ポーリング内蔵)
vast execute <ID> "<COMMAND>"             インスタンス上でのコマンド直接実行
```

### アカウント & セキュリティ
```bash
vast set api-key <KEY>            API キーの保存 (~/.config/vastai/vast_api_key)
vast reset api-key                マスター API キーのリセット
vast show api-keys                API キー一覧表示
vast create api-key --name <NAME> 制限付き API キーの作成
vast delete api-key <ID>          API キーの削除
vast show user                    アカウント情報・クレジット残高表示
vast show audit-logs              アカウント操作履歴
vast show env-vars                環境変数シークレット一覧
vast create env-var <KEY> <VAL>   環境変数の設定
vast delete env-var <KEY>         環境変数の削除
```

### グローバルフラグ
* `--raw`: すべてのコマンドで機械可読な整形 JSON を標準出力
* `--explain`: 送信される HTTP リクエスト (URL, Headers, Body) の生データを表示
* `--curl`: 実行と同等の `curl` コマンドを表示して終了
* `--url <URL>`: Vast.ai サーバーの REST API URL を上書き
* `--retry <N>`: 一時的な HTTP 429/502/503/504 エラー時のリトライ上限 (デフォルト: 3)
* `--version`: CLI バージョン表示
* `-h, --help`: ヘルプ表示

---

## アーキテクチャ設計

1. **認証解決パイプライン ([`pkg/config`](pkg/config/))**:
   CLI 引数 `--api-key` → 環境変数 `VAST_API_KEY` → `$XDG_CONFIG_HOME/vastai/vast_api_key` (`~/.config/vastai/vast_api_key`) → レガシー `~/.vast_api_key` の順序でフォールバック探索。
2. **クエリ構文解析 ([`pkg/query`](pkg/query/))**:
   `gpu_name=RTX_5090 num_gpus>=1` などの文字列式をパース。アンダースコアの半角空白置換、エイリアス解決、単位変換（GB→MB）を自動適用。
3. **HTTP 通信・リトライ ([`pkg/client`](pkg/client/))**:
   Go 標準の `net/http` を使用し、接続エラーやレート制限時の指数バックオフリトライ、および非同期結果 URL の自動ポーリングを内蔵。
4. **表示フォーマット ([`pkg/display`](pkg/display/))**:
   公式 CLI と完全互換の 3 ブロック表形式表示と、スクリプト連携・パイプライン用の `--raw` JSON 出力をサポート。

---

## ライセンス

MIT License
