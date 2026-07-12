# Discord Lolipop Bot

ロリポップ！for Gamers(VPS)上のゲームサーバーを、DiscordのスラッシュコマンドからSSH経由で操作するBot。
友人向けに仲間内で使用する用途で作成したため、不特定多数の参加者がいるサーバーでは使用しないこと。

## 機能

SSHでリモートサーバーに接続し`sudo /usr/local/bin/game <action>`を実行、結果をDiscordに返す。
(すべて実行者本人にのみ表示)

- `/start-server`: サーバー起動
- `/stop-server`: サーバー停止
- `/restart-server`: サーバー再起動
- `/status-server`: サーバー状態確認

## 前提条件

- Go 1.25
- Discord Bot Token
- SSH鍵認証でログインできるロリポップ！for Gamersサーバー
- サーバー側で、Botが使うSSHユーザーが`sudo /usr/local/bin/game`をパスワードなしで実行できること
  (`/etc/sudoers.d/`に`NOPASSWD`エントリが必要)

## セットアップ

### Discord Botの作成

1. [Discord Developer Portal](https://discord.com/developers/applications)でアプリケーションを作成
2. Botタブから「Add Bot」をクリックしBot Tokenを取得
3. OAuth2タブで`bot`/`applications.commands`スコープを選択して招待

### 環境変数の設定

```bash
cp .env.example .env
```

`.env`を編集。

- `DISCORD_BOT_TOKEN`: Discord Bot Token
- `SSH_HOST`/`SSH_PORT`/`SSH_USER`: 接続先サーバー情報
- `SSH_KEY_PATH`または`SSH_PRIVATE_KEY`: どちらか一方
  - ローカル実行時は`SSH_KEY_PATH`に鍵ファイルパスを指定
  - ホスティング環境等でファイルを配置できない場合は`SSH_PRIVATE_KEY`に鍵の中身をそのまま設定(ホスティング先のシークレット機能を利用)
- `SSH_KEY_PASSPHRASE`: パスフレーズ付き鍵の場合のみ
- `SSH_KNOWN_HOSTS`: known_hostsファイルのパス。未設定の場合ホスト鍵検証が無効化(本番では設定推奨)

### 実行権限の設定

```bash
cp permissions.example.json permissions.json
```

`permissions.json`でコマンド実行を許可するモードを選択。

```json
{
  "mode": "allow",
  "ids": ["DiscordのユーザーID"]
}
```

- `mode: "open"`: 誰でも実行可能(`ids`は無視)
- `mode: "allow"`: `ids`に含まれるユーザーのみ実行可能
- `mode: "deny"`: `ids`に含まれるユーザー以外が実行可能

### 起動方法

#### ローカルで起動

```bash
go run ./cmd/lolipop-bot
```

またはビルドして実行:

```bash
go build -o lolipop-bot ./cmd/lolipop-bot
./lolipop-bot
```

#### Dockerで起動

`.env`と`permissions.json`を用意した上で:

```bash
docker compose up -d
```

`SSH_KEY_PATH`はホスト側パスなのでコンテナ内では使えない。`compose.yml`側で鍵ファイル・known_hostsをコンテナ内の固定パス(`/app/ssh_key`、`/app/known_hosts`)にマウントし、`SSH_KEY_PATH`/`SSH_KNOWN_HOSTS`をそのパスで上書きしている。鍵ファイルの場所を変える場合は`compose.yml`のマウント元を書き換える。
