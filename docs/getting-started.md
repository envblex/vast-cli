# Vast.ai 初心者向けスタートガイド

このガイドでは、**Vast.ai を一度も使ったことがない方**を対象に、アカウント開設から GPU インスタンスの起動、SSH 接続、そして使い終わった後のインスタンス破棄までをゼロからステップバイステップで解説します。

> **対応公式バージョン**: Vast.ai 公式 CLI `vastai v1.8.3`（2026年最新）の仕様に基づいています。

---

## 目次
1. [Vast.ai とは？](#1-vastai-とは)
2. [事前準備 (アカウント開設・チャージ・APIキー)](#2-事前準備)
3. [SSH 鍵の準備と登録](#3-ssh-鍵の準備と登録)
4. [GPU オファーの検索](#4-gpu-オファーの検索)
5. [インスタンスの作成と起動](#5-インスタンスの作成と起動)
6. [SSH 接続とファイル転送](#6-ssh-接続とファイル転送)
7. [【最重要】課金を止める (Stop と Destroy の違い)](#7-最重要課金を止める-stop-と-destroy-の違い)
8. [よくあるトラブルと対処法](#8-よくあるトラブルと対処法)

---

## 1. Vast.ai とは？

Vast.ai は、世界中のデータセンターや個人の GPU サーバーを直接時間貸しで借りられる **GPU クラウドマーケットプレイス** です。

* **圧倒的な安さ**: AWS や GCP、Azure などの大手クラウドと比較して **1/3 〜 1/5 程度の価格**（RTX 4090 が $0.3〜$0.5/時間 前後、RTX 5090 が $0.5〜$0.6/時間 前後）で利用できます。
* **Docker ベース**: PyTorch、CUDA、vLLM、ComfyUI などの Docker コンテナとして瞬時に起動します。
* **従量課金**: 1 秒単位で課金され、不要になったら即破棄できます。

---

## 2. 事前準備

### ステップ 1: アカウント作成
1. [https://cloud.vast.ai](https://cloud.vast.ai) にアクセスし、サインアップします。

### ステップ 2: クレジットのチャージ
Vast.ai はプリペイド（事前チャージ）方式です。
1. コンソールの [Billing](https://cloud.vast.ai/billing/) に進みます。
2. クレジットカードまたは暗号通貨でデポジットを入金します（最初はテスト用に **$5〜$10** 程度で十分です）。

### ステップ 3: API キーの取得と CLI への登録
1. [Manage Keys](https://cloud.vast.ai/manage-keys/) 画面を開きます。
2. 表示されている API Key をコピーします。
3. ローカルのターミナルで次のコマンドを実行して登録します：
   ```bash
   vast set api-key <コピーしたAPIキー>
   ```
4. アカウント情報と残高が表示されるか確認します：
   ```bash
   vast show user
   ```

---

## 3. SSH 鍵の準備と登録

インスタンスに安全にログインするために、SSH 公開鍵を登録します。

### 鍵をお持ちでない場合
```bash
ssh-keygen -t ed25519 -C "vast-ai"
# (すべて Enter を押せば ~/.ssh/id_ed25519 が生成されます)
```

### Vast.ai に SSH 公開鍵を登録
```bash
# 公開鍵ファイルを指定して登録
vast create ssh-key ~/.ssh/id_ed25519.pub

# 登録された鍵の確認
vast show ssh-keys
```

> [!IMPORTANT]
> **インスタンスを作成する前に必ず SSH 鍵を登録してください。**
> 鍵を登録せずに作成したインスタンスには SSH ログインできなくなります。

---

## 4. GPU オファーの検索

条件を指定して、借りたい GPU の出品（オファー）を探します。

```bash
# 例 1: RTX 4090 × 1枚、信頼性95%以上、直結ポートあり、DL性能/$ 順
vast search offers "gpu_name=RTX_4090 num_gpus=1 reliability>0.95 direct_port_count>=1" -o "dlperf_usd-" --limit 5

# 例 2: 最新の RTX 5090 を探す
vast search offers "gpu_name=RTX_5090 num_gpus=1" --limit 5

# 例 3: 予算上限 $0.5/時間 で探す
vast search offers "dph<=0.5 num_gpus>=1" --limit 5
```

出力結果の先頭列にある **`ID`**（例: `49820767`）を控えます。

---

## 5. インスタンスの作成と起動

控えたオファー ID を指定してインスタンスを作成します。

```bash
vast create instance <OFFER_ID> \
  --image pytorch/pytorch:2.4.0-cuda12.4-cudnn9-runtime \
  --disk 30 \
  --ssh \
  --direct
```

* `--image`: 動かしたい Docker イメージ名。
* `--disk`: ディスク容量 (GB 単位)。
* `--ssh`: SSH 接続を有効化。
* `--direct`: 高速な直接ポートマッピング接続を使用。

### 起動ステータスの確認
作成したら、インスタンスが起動するまで状態を確認します：

```bash
vast show instances
```

* `actual_status` の推移：
  * `created`: インスタンス初期化中
  * `loading`: Docker イメージをダウンロード中
  * `running`: **起動完了！課金と SSH 接続が可能になります**

---

## 6. SSH 接続とファイル転送

### SSH ログイン
インスタンスが `running` になったら、接続文字列を取得します：

```bash
vast ssh-url <INSTANCE_ID>
# 出力例: ssh://root@192.0.2.1:34567
```

そのままワンライナーで SSH 接続できます：
```bash
# ssh-url の出力を整形して接続
ssh root@$(vast ssh-url <INSTANCE_ID> | sed 's|ssh://root@||; s|:| -p |')
```

ログイン後、`nvidia-smi` を叩けば GPU が即座に認識されていることが確認できます。

### ファイル転送 (ローカル ⇔ インスタンス)
`scp` や `rsync` を使用します：

```bash
# scp URL の確認
vast scp-url <INSTANCE_ID>
# 出力例: scp://root@192.0.2.1:34567

# ローカルのファイルをインスタンスへアップロード
scp -P 34567 ./my_script.py root@192.0.2.1:/workspace/

# インスタンスの生成物をローカルへダウンロード
scp -P 34567 root@192.0.2.1:/workspace/result.pt ./
```

---

## 7. 【最重要】課金を止める (Stop と Destroy の違い)

クラウド GPU で最も初心者が陥りやすいトラブルが **「使い終わったのに放置して課金が続く」** ことです。
2 つの停止方法の違いを必ず理解してください：

| 操作 | コマンド | GPU 料金 | ストレージ料金 | データの保持 | 用途 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **一時停止 (Stop)** | `vast stop instance <ID>` | **停止 ($0)** | **発生し続ける** | ディスク保持 | 後ですぐ再開する場合 |
| **完全破棄 (Destroy)** | `vast destroy instance <ID> -y` | **停止 ($0)** | **完全停止 ($0)** | 完全に消去 | **作業が完了した時** |

> [!CAUTION]
> 作業が終わったら、必ず **`vast destroy instance <ID> -y`** を実行してインスタンスを破棄してください。
> `stop` のままだと、GPU は使っていなくてもディスク保管料が残高から引き落とされ続けます。

```bash
# インスタンスを完全に破棄して課金をゼロにする
vast destroy instance <INSTANCE_ID> -y

# 破棄されたことを確認 (一覧が空になること)
vast show instances
```

---

## 8. よくあるトラブルと対処法

### Q1. `actual_status` が `loading` のまま進まない / `exited` になった
* **原因**: ホストのネット回線が遅い、またはホスト側で Docker の起動に失敗した。
* **対処**: 10分以上待っても動かない場合や `exited`/`offline` になった場合は、`vast destroy instance <ID> -y` で一度破棄し、別のホストのオファー（reliability > 0.98 のもの）を選び直してください。

### Q2. SSH 接続しようとすると `Permission denied (publickey)` になる
* **原因**: インスタンス作成時に SSH 公開鍵が Vast.ai に登録されていなかった。
* **対処**: インスタンス作成後に登録した鍵は自動反映されません。`vast attach ssh <INSTANCE_ID> "$(cat ~/.ssh/id_ed25519.pub)"` で後から追加するか、一度破棄して鍵登録後に作り直してください。

### Q3. 突然インスタンスが停止した
* **原因**: `--bid_price` を使って「Spot（割り込み可能）インスタンス」を借りていた場合、他のユーザーに高値で入札されて奪われた可能性があります。
* **対処**: 途中で止められたくない本番学習では、通常の On-demand（デフォルト）で借りることをおすすめします。
