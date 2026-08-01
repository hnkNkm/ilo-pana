# ilo-pana GUI Architecture Plan

## Overview
ilo-pana (「良いツール」) のGUI版をWails + Svelteで構築する計画書。
既存のCLI版コアロジックを再利用し、デスクトップアプリケーションとして提供。

## 技術スタック

### Backend (Go)
- **Framework**: Wails v2.11.0
- **Core Logic**: 既存の`internal/`パッケージを再利用
  - `internal/client` - HTTPクライアント
  - `internal/request` - リクエスト構築
  - `internal/response` - レスポンス処理  
  - `internal/session` - セッション管理
  - `internal/variables` - 変数展開
  - `internal/env` - 環境ファイル

### Frontend
- **Framework**: Svelte 5
- **Language**: TypeScript
- **UI Components**: shadcn-svelte (Bits UI/Melt UI + Tailwind CSS)
- **State Management**:
  - Server State: TanStack Query for Svelte
  - Client State: Svelte stores (built-in)
- **Build Tool**: Vite 5
- **Package Manager**: pnpm

## プロジェクト構造

```
api-tester/
├── internal/           # 共有コアロジック（CLI/GUI共通）
│   ├── client/
│   ├── request/
│   ├── response/
│   ├── session/
│   ├── variables/
│   └── env/
├── cmd/               # CLI版
│   └── ilo-pana/
│       └── main.go
├── gui/               # GUI版（Wails）
│   ├── app.go         # Wailsバックエンドブリッジ
│   ├── main.go        # エントリーポイント
│   ├── embed.go       # フロントエンドアセット埋め込み
│   ├── wails.json     # Wails設定
│   ├── build/         # ビルド出力
│   └── frontend/      # Svelteフロントエンド
│       ├── src/
│       │   ├── lib/
│       │   │   ├── components/
│       │   │   │   ├── RequestBuilder.svelte
│       │   │   │   ├── ResponseViewer.svelte
│       │   │   │   ├── CollectionExplorer.svelte
│       │   │   │   ├── EnvironmentManager.svelte
│       │   │   │   └── ui/  # shadcn-svelte
│       │   │   ├── stores/
│       │   │   │   ├── request.ts
│       │   │   │   ├── collections.ts
│       │   │   │   └── environment.ts
│       │   │   └── wailsjs/  # Wails生成バインディング
│       │   ├── App.svelte
│       │   └── main.ts
│       ├── package.json
│       ├── pnpm-lock.yaml
│       ├── vite.config.ts
│       ├── tsconfig.json
│       └── tailwind.config.js
├── docs/              # ドキュメント
├── go.mod            # 共通依存関係
├── go.sum
├── README.md
└── .gitignore
```

## 主要コンポーネント

### 1. RequestBuilder
- HTTPメソッド選択（GET, POST, PUT, DELETE, etc.）
- URL入力（変数展開対応 {{variable}}）
- ヘッダー管理
- ボディエディタ（JSON/Form/Raw）
- クエリパラメータ管理

### 2. ResponseViewer  
- ステータスコード表示
- レスポンスヘッダー表示（機密情報マスキング）
- レスポンスボディ表示（JSON整形）
- 実行時間表示
- レスポンスサイズ表示

### 3. CollectionExplorer
- フォルダ構造でリクエスト管理
- コレクションのインポート/エクスポート
- リクエストの検索
- お気に入り機能

### 4. EnvironmentManager
- 環境変数の作成/編集
- .envファイルのインポート
- 環境の切り替え（dev/staging/prod）

### 5. SessionManager
- セッション一覧表示
- Cookie管理
- セッション永続化

## Wails Backend API

```go
// gui/app.go

type App struct {
    ctx    context.Context
    client *client.Client
}

// ExecuteRequest - HTTPリクエストを実行
func (a *App) ExecuteRequest(cfg RequestConfig) (*ResponseData, error)

// LoadEnvironment - 環境ファイル読み込み
func (a *App) LoadEnvironment(filename string) error

// GetSessions - セッション一覧取得
func (a *App) GetSessions() ([]SessionInfo, error)

// SaveCollection - コレクション保存
func (a *App) SaveCollection(collection Collection) error
```

## Frontend State Management

```typescript
// Svelte stores - クライアント状態
import { writable, derived } from 'svelte/store';

// 現在のリクエスト
export const activeRequest = writable({
    method: 'GET',
    url: '',
    headers: {},
    body: null
});

// 環境変数
export const environment = writable({
    name: 'development',
    variables: {}
});

// TanStack Query - サーバー状態
import { createQuery } from '@tanstack/svelte-query';

export function useExecuteRequest(config) {
    return createQuery({
        queryKey: ['request', config],
        queryFn: () => window.go.main.App.ExecuteRequest(config)
    });
}
```

## 開発フェーズ

### Phase 1: 基盤構築（1週間）
- [ ] Wails プロジェクト初期化
- [ ] Svelte + TypeScript セットアップ
- [ ] pnpm によるパッケージ管理
- [ ] shadcn-svelte 導入
- [ ] TailwindCSS 設定
- [ ] 基本レイアウト実装

### Phase 2: コア機能（2週間）
- [ ] RequestBuilder コンポーネント
- [ ] ResponseViewer コンポーネント
- [ ] Wails バックエンドAPI実装
- [ ] 既存Go コアとの統合
- [ ] 変数展開機能

### Phase 3: 高度な機能（2週間）
- [ ] CollectionExplorer 実装
- [ ] EnvironmentManager 実装
- [ ] SessionManager 統合
- [ ] インポート/エクスポート機能
- [ ] ホットキー対応

### Phase 4: 品質向上（1週間）
- [ ] エラーハンドリング強化
- [ ] パフォーマンス最適化
- [ ] E2Eテスト追加
- [ ] ドキュメント作成
- [ ] リリースビルド設定

## ビルド & デプロイ

### 開発
```bash
cd gui
wails dev
```

### ビルド
```bash
# macOS
wails build -platform darwin/amd64,darwin/arm64

# Windows
wails build -platform windows/amd64

# Linux  
wails build -platform linux/amd64
```

### リリース
- GitHub Actions でマルチプラットフォームビルド
- GitHub Releases で配布
- Homebrew (macOS)
- Chocolatey (Windows)
- AppImage/Snap (Linux)

## 差別化ポイント

1. **軽量・高速**: Wails使用でElectronより軽量
2. **既存CLIとの統合**: 同一コアロジック使用
3. **オフライン対応**: ローカルファーストの設計
4. **Git連携**: コレクションをGit管理可能
5. **日本語対応**: UI/UXの日本語最適化

## 参考資料

- [Wails Documentation](https://wails.io/docs)
- [Svelte 5 Documentation](https://svelte.dev/docs)
- [shadcn-svelte](https://www.shadcn-svelte.com/)
- [TanStack Query for Svelte](https://tanstack.com/query/latest/docs/framework/svelte/overview)