
# Payment 微服務

這是一個用於整合 Stripe 支付功能的微服務，提供全面的支付解決方案，包括客戶管理、產品和價格管理、訂閱、一鍵購買、退款處理和發票管理等功能。本微服務使用 Go 語言實現，並使用 gRPC 進行 API 通訊。

## 目錄
- [安裝與設置](#安裝與設置)
- [環境變數](#環境變數)
- [gRPC 服務](#grpc-服務)
- [數據庫設計](#數據庫設計)
- [API 設計](#api-設計)
- [Webhook 處理](#webhook-處理)
- [安全考慮](#安全考慮)
- [測試](#測試)
- [運行指令](#運行指令)
- [貢獻指南](#貢獻指南)

## 安裝與設置

1. **克隆代碼庫**：
    ```bash
    git clone git@github.com:flare-go/payment.git
    cd payment
    ```

2. **安裝依賴**：
   使用 Go modules 安裝所需依賴
    ```bash
    go mod tidy
    ```

3. **設置環境變數**：
   創建 `.env` 文件，並添加必要的環境變數。環境變數的詳情請參閱[環境變數](#環境變數)部分。

## 環境變數

| 變數名             | 描述                      |
|-------------------|-------------------------|
| `STRIPE_API_KEY`  | 您的 Stripe API 密鑰       |
| `DATABASE_URL`    | 數據庫連接 URL           |
| `PORT`            | gRPC 服務運行的端口        |

## gRPC 服務

`payment` 微服務使用 gRPC 進行通訊。以下是支持的 gRPC 方法：

### 客戶管理

- `CreateCustomer`: 創建一個新的客戶
- `GetCustomer`: 根據 ID 獲取客戶信息
- `UpdateCustomer`: 更新客戶信息

### 產品管理

- `CreateProduct`: 創建一個新的產品
- `GetProduct`: 根據 ID 獲取產品信息
- `UpdateProduct`: 更新產品信息
- `ListProducts`: 列出所有產品

### 價格管理

- `CreatePrice`: 創建一個新的價格
- `GetPrice`: 根據 ID 獲取價格信息
- `UpdatePrice`: 更新價格信息
- `ListPrices`: 列出所有價格

### 訂閱管理

- `CreateSubscription`: 創建一個新的訂閱
- `GetSubscription`: 根據 ID 獲取訂閱信息
- `UpdateSubscription`: 更新訂閱信息
- `CancelSubscription`: 取消訂閱
- `ListSubscriptions`: 列出所有訂閱

### 支付意圖管理

- `CreatePaymentIntent`: 創建一個新的支付意圖
- `GetPaymentIntent`: 根據 ID 獲取支付意圖信息
- `ConfirmPaymentIntent`: 確認支付意圖
- `CancelPaymentIntent`: 取消支付意圖

### 退款處理

- `CreateRefund`: 創建退款
- `GetRefund`: 根據 ID 獲取退款信息

### 發票管理

- `GetInvoice`: 根據 ID 獲取發票信息
- `ListInvoices`: 列出所有發票
- `PayInvoice`: 支付發票

### 支付方式管理

- `CreatePaymentMethod`: 添加支付方式
- `GetPaymentMethod`: 根據 ID 獲取支付方式信息
- `UpdatePaymentMethod`: 更新支付方式信息
- `DeletePaymentMethod`: 刪除支付方式
- `ListPaymentMethods`: 列出所有支付方式

### Webhook 處理

- `HandleWebhook`: 處理來自 Stripe 的 Webhook 事件

## 數據庫設計

使用 PostgreSQL 作為數據庫，並設計了多張表來存儲相關的支付信息。

### 主要表結構

- **customers**: 儲存客戶信息
- **products**: 儲存產品信息
- **prices**: 儲存價格信息
- **subscriptions**: 儲存訂閱信息
- **invoices**: 儲存發票信息
- **payment_methods**: 儲存支付方式信息
- **payment_intents**: 儲存支付意圖信息

詳細的數據庫結構請參閱 `sql/schema.sql` 文件。

## API 設計

微服務的 API 使用 gRPC 通訊，具體方法和數據結構參見 `proto/payment.proto` 文件。

## Webhook 處理

本服務通過 `HandleWebhook` 方法來處理來自 Stripe 的 Webhook 事件。支持的事件類型包括：
- `payment_intent.succeeded`
- `invoice.paid`
- `customer.subscription.deleted`

Webhook 事件處理會自動同步 Stripe 的狀態到本地數據庫中。

## 安全考慮

1. **使用 HTTPS**：所有的 gRPC 通訊應使用安全的 HTTPS 通道。
2. **API 密鑰保護**：確保您的 Stripe API 密鑰存儲在安全的地方。
3. **數據加密**：敏感數據如客戶信息應該在存儲和傳輸時進行加密。
4. **角色權限控制**：對每個 API 端點實現適當的角色權限控制。

## 測試

為了確保微服務的可靠性，請運行單元測試和集成測試：
```bash
go test ./...
```

## 運行指令

運行服務：
```bash
go run main.go
```

或者使用 Docker 運行：
```bash
docker build -t payment-service .
docker run -p 8080:8080 payment-service
```

## 貢獻指南

歡迎貢獻！請遵循以下步驟提交貢獻：
1. Fork 本倉庫
2. 創建您的功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的修改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打開一個 Pull Request

感謝您的貢獻！

## 聯絡方式

如有任何問題或建議，請聯繫 [your-email@example.com](mailto:your-email@example.com)。
