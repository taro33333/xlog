# xlog

Go標準ライブラリ `log/slog` をベースにした、高性能・ゼロ依存のロギングライブラリ。

## 特徴

- **外部依存ゼロ**: Go標準ライブラリ (`log/slog`) のみで構築
- **Context伝播**: TraceID、UserID、RequestIDをcontextから自動抽出
- **環境対応**: 開発環境（カラーテキスト）と本番環境（JSON）を一行で切り替え
- **Functional Optionsパターン**: 拡張性の高い設定
- **正確な呼び出し元情報**: ラッパー関数によるスタックズレなし
- **スレッドセーフ**: 並行処理に完全対応
- **高パフォーマンス**: メモリ割り当てを最小化

## インストール

```bash
go get github.com/taro33333/xlog
```

## クイックスタート

```go
package main

import (
    "context"
    "log/slog"

    "github.com/taro33333/xlog"
)

func main() {
    // ロガー初期化（デフォルトは開発モード）
    xlog.Init()

    ctx := context.Background()
    xlog.Info(ctx, "サーバー起動", "port", 8080)
}
```

## 設定

### 環境ベースの設定

```go
// 開発環境: カラーテキスト出力
xlog.Init(
    xlog.WithEnvironment(xlog.Development),
    xlog.WithLevel(slog.LevelDebug),
)

// 本番環境: JSON構造化ログ
xlog.Init(
    xlog.WithEnvironment(xlog.Production),
    xlog.WithLevel(slog.LevelInfo),
)
```

### 利用可能なオプション

| オプション | 説明 | デフォルト値 |
|------------|------|--------------|
| `WithEnvironment(env)` | ログ環境を設定 | `Development` |
| `WithLevel(level)` | 最小ログレベルを設定 | `slog.LevelInfo` |
| `WithOutput(w)` | 出力先を設定 | `os.Stdout` |
| `WithSource(bool)` | ソース位置の有効/無効 | `true` |
| `WithTimeFormat(fmt)` | 時刻フォーマット（開発モード） | `time.RFC3339` |
| `WithContextKeys(keys...)` | 抽出するContextキーを設定 | TraceID, UserID, RequestID |

## Context伝播

xlogはcontextから値を自動抽出し、ログ出力に追加します。

```go
// トレース情報をcontextに追加
ctx := context.Background()
ctx = xlog.WithTraceID(ctx, "abc-123-def")
ctx = xlog.WithUserID(ctx, "user-456")
ctx = xlog.WithRequestID(ctx, "req-789")

// ログにtrace_id、user_id、request_idが自動付加される
xlog.Info(ctx, "リクエスト処理中", "action", "create")
```

### カスタムContextキー

```go
// カスタムContextキーを定義
const MyCustomKey xlog.ContextKey = "custom_field"

// カスタムキーで初期化
xlog.Init(
    xlog.WithContextKeys(
        xlog.TraceIDKey,
        xlog.UserIDKey,
        MyCustomKey,
    ),
)

// contextに追加
ctx = xlog.WithContext(ctx, MyCustomKey, "custom_value")
```

### 定義済みContextキー

| キー | 説明 |
|------|------|
| `xlog.TraceIDKey` | 分散トレーシングID |
| `xlog.UserIDKey` | ユーザー識別子 |
| `xlog.RequestIDKey` | リクエスト識別子 |
| `xlog.SessionIDKey` | セッション識別子 |
| `xlog.SpanIDKey` | スパン識別子 |

## ログAPI

すべてのログ関数は第一引数に `context.Context` を取ります：

```go
xlog.Debug(ctx, "デバッグメッセージ", "key", "value")
xlog.Info(ctx, "情報メッセージ", "key", "value")
xlog.Warn(ctx, "警告メッセージ", "key", "value")
xlog.Error(ctx, "エラーメッセージ", "err", err)
```

### Loggerインスタンス

```go
// デフォルトロガーを取得
logger := xlog.Default()

// 属性付きロガーを作成
logger = xlog.With("service", "api", "version", "1.0")

// グループ付きロガーを作成
logger = xlog.WithGroup("http")
logger.Info(ctx, "リクエスト受信", "method", "GET")
```

## 出力例

### 開発モード

```
2024-01-15 10:30:45 INF main.go:25 サーバー起動 port=8080
2024-01-15 10:30:46 INF handler.go:42 リクエスト処理中 trace_id=abc-123 user_id=user-456 action=create
2024-01-15 10:30:47 ERR handler.go:55 処理失敗 err="connection refused"
```

### 本番モード（JSON）

```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","source":{"file":"main.go","line":25},"msg":"サーバー起動","port":8080}
{"time":"2024-01-15T10:30:46Z","level":"INFO","source":{"file":"handler.go","line":42},"msg":"リクエスト処理中","trace_id":"abc-123","user_id":"user-456","action":"create"}
```

## 標準ライブラリとの統合

xlogは標準 `log` パッケージからの出力をリダイレクトします：

```go
import "log"

xlog.Init()

// xlogでキャプチャされる
log.Println("標準logからのメッセージ")
```

## HTTPミドルウェアの例

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // リクエストIDを抽出または生成
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        // contextに追加
        ctx := r.Context()
        ctx = xlog.WithRequestID(ctx, requestID)
        ctx = xlog.WithTraceID(ctx, r.Header.Get("X-Trace-ID"))

        xlog.Info(ctx, "リクエスト開始",
            "method", r.Method,
            "path", r.URL.Path,
        )

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## パフォーマンス

xlogは高負荷環境向けに設計されています：

- ログフォーマット用の事前割り当てバッファ
- interface{}ボクシングの最小化
- 効率的なcontext値抽出
- 書き込み操作のみに sync.Mutex を使用

### ベンチマーク結果

```
BenchmarkInfo-8           1364210   1006 ns/op   697 B/op   7 allocs/op
BenchmarkInfoParallel-8   3071288   459.6 ns/op  653 B/op   7 allocs/op
```

## スレッドセーフティ

xlogは完全にスレッドセーフです。すべてのエクスポート関数およびメソッドは、複数のgoroutineから安全に呼び出せます。

## ライセンス

MIT License
