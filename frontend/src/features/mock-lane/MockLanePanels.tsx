import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  calculateCartTotals,
  getHistoryDetail,
  getProductDetail,
  listHistory,
  listProducts,
  resolveProductForCart,
  type MockCartItem,
  type MockOrderDetail,
  type MockOrderSummary,
  type MockProduct,
} from "./mockApi";

export function MockLanePanels() {
  const [loginId, setLoginId] = useState("c-1");
  const [password, setPassword] = useState("");
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [currentCustomerID, setCurrentCustomerID] = useState("c-1");
  const [loginMessage, setLoginMessage] = useState("");

  const [products, setProducts] = useState<MockProduct[]>([]);
  const [keyword, setKeyword] = useState("");
  const [productDetailMessage, setProductDetailMessage] = useState("");
  const [productDetail, setProductDetail] = useState<MockProduct | null>(null);

  const [cartMessage, setCartMessage] = useState("");
  const [cartItems, setCartItems] = useState<MockCartItem[]>([]);
  const [draftQuantities, setDraftQuantities] = useState<Record<string, string>>({});

  const [historyMessage, setHistoryMessage] = useState("");
  const [historySummaries, setHistorySummaries] = useState<MockOrderSummary[]>([]);
  const [historyDetail, setHistoryDetail] = useState<MockOrderDetail | null>(null);

  useEffect(() => {
    void listProducts().then((items) => {
      setProducts(items);
      const defaults = items.reduce<Record<string, string>>((acc, item) => {
        acc[item.sku] = "1";
        return acc;
      }, {});
      setDraftQuantities(defaults);
    });
  }, []);

  const activeProducts = useMemo(() => {
    const normalized = keyword.trim().toLowerCase();
    return products.filter(
      (product) =>
        product.isActive &&
        (normalized === "" ||
          product.name.toLowerCase().includes(normalized) ||
          product.sku.toLowerCase().includes(normalized)),
    );
  }, [products, keyword]);
  const cartTotals = useMemo(() => calculateCartTotals(cartItems), [cartItems]);

  function onLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoginMessage("");
    if (loginId.trim() === "" || password.trim() === "") {
      setLoginMessage("ログイン情報を入力してください");
      return;
    }
    setCurrentCustomerID(loginId.trim());
    setIsLoggedIn(true);
  }

  async function onLoadProductDetail(sku: string) {
    setProductDetailMessage("");
    setProductDetail(null);
    const result = await getProductDetail(sku);
    if (result.kind === "success") {
      setProductDetail(result.product);
      return;
    }
    setProductDetailMessage("商品が見つかりません (404)");
  }

  function onAddCartItem(sku: string) {
    setCartMessage("");

    const quantityRaw = draftQuantities[sku] ?? "1";
    const quantity = Number(quantityRaw);
    if (!Number.isInteger(quantity) || quantity < 1) {
      setCartMessage("カート追加に失敗しました (400)");
      return;
    }
    const product = resolveProductForCart(sku);
    if (!product) {
      setCartMessage("カート追加に失敗しました (400)");
      return;
    }

    setCartItems((current) => {
      const existing = current.find((item) => item.sku === sku);
      if (!existing) {
        return [
          ...current,
          {
            sku: product.sku,
            name: product.name,
            quantity,
            unitPrice: product.unitPrice,
            lineAmount: product.unitPrice * quantity,
          },
        ];
      }
      return current.map((item) =>
        item.sku !== sku
          ? item
          : {
              ...item,
              quantity: item.quantity + quantity,
              lineAmount: item.unitPrice * (item.quantity + quantity),
            },
      );
    });
    setCartMessage("カートに追加しました");
  }

  function onUpdateQuantity(sku: string) {
    const raw = draftQuantities[sku] ?? "";
    const quantity = Number(raw);
    if (!Number.isInteger(quantity) || quantity < 1) {
      setCartMessage("数量変更に失敗しました (400)");
      return;
    }

    setCartItems((current) =>
      current.map((item) =>
        item.sku !== sku
          ? item
          : { ...item, quantity, lineAmount: item.unitPrice * quantity },
      ),
    );
    setCartMessage("数量を更新しました");
  }

  function onRemoveCartItem(sku: string) {
    setCartItems((current) => current.filter((item) => item.sku !== sku));
    setCartMessage("商品を削除しました");
  }

  async function onLoadHistory() {
    setHistoryMessage("");
    setHistoryDetail(null);

    const customerId = currentCustomerID.trim();
    if (customerId === "") {
      setHistoryMessage("入力内容を確認してください");
      return;
    }

    const summaries = await listHistory(customerId);
    setHistorySummaries(summaries);
  }

  async function onLoadHistoryDetail(orderId: string) {
    const customerId = currentCustomerID.trim();
    const result = await getHistoryDetail(customerId, orderId);
    if (result.kind === "success") {
      setHistoryDetail(result.order);
      setHistoryMessage("");
      return;
    }
    setHistoryDetail(null);
    setHistoryMessage("履歴詳細が見つかりません");
  }

  return (
    <section className="panel panel-mock">
      <h2>ECフロント仮実装（mock）</h2>

      {!isLoggedIn ? (
        <section className="subpanel">
          <h3>ログイン（仮）</h3>
          <p>認証API未実装のため、入力値をそのまま会員IDとして扱います。</p>
          <form onSubmit={onLogin} className="stack-form">
            <label htmlFor="loginId">会員ID</label>
            <input
              id="loginId"
              name="loginId"
              value={loginId}
              onChange={(event) => setLoginId(event.target.value)}
            />
            <label htmlFor="password">パスワード</label>
            <input
              id="password"
              name="password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
            <button type="submit">ログイン</button>
          </form>
          {loginMessage !== "" ? (
            <p role="status" className="status-message">
              {loginMessage}
            </p>
          ) : null}
        </section>
      ) : null}

      {isLoggedIn ? (
        <>
          <section className="subpanel">
            <h3>商品一覧</h3>
            <p>ログイン中: {currentCustomerID}</p>
            <div className="product-toolbar">
              <label htmlFor="productKeyword">検索</label>
              <input
                id="productKeyword"
                name="productKeyword"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="商品名 / SKU"
              />
            </div>

            <ul className="product-grid">
              {activeProducts.map((product) => (
                <li key={product.sku} className="product-card">
                  <p className="product-name">{product.name}</p>
                  <p>SKU: {product.sku}</p>
                  <p>
                    価格: {product.unitPrice} {product.currency}
                  </p>
                  <label htmlFor={`product-qty-${product.sku}`}>数量</label>
                  <input
                    id={`product-qty-${product.sku}`}
                    value={draftQuantities[product.sku] ?? "1"}
                    onChange={(event) =>
                      setDraftQuantities((current) => ({
                        ...current,
                        [product.sku]: event.target.value,
                      }))
                    }
                  />
                  <div className="actions-row">
                    <button type="button" onClick={() => void onLoadProductDetail(product.sku)}>
                      詳細
                    </button>
                    <button type="button" onClick={() => onAddCartItem(product.sku)}>
                      カートに追加
                    </button>
                  </div>
                </li>
              ))}
            </ul>
            {activeProducts.length === 0 ? <p>該当商品がありません</p> : null}
          </section>

          <section className="subpanel">
            <h3>商品詳細</h3>
            {productDetail ? (
              <article className="result-card">
                <p>SKU: {productDetail.sku}</p>
                <p>商品名: {productDetail.name}</p>
                <p>
                  価格: {productDetail.unitPrice} {productDetail.currency}
                </p>
              </article>
            ) : (
              <p>商品を選択すると詳細を表示します。</p>
            )}
            {productDetailMessage !== "" ? (
              <p role="status" className="status-message">
                {productDetailMessage}
              </p>
            ) : null}
          </section>

          <section className="subpanel">
            <h3>カート</h3>
            <ul>
              {cartItems.map((item) => (
                <li key={item.sku}>
                  {item.sku} x {item.quantity} = {item.lineAmount}
                  <label htmlFor={`cart-qty-${item.sku}`}>数量変更</label>
                  <input
                    id={`cart-qty-${item.sku}`}
                    value={draftQuantities[item.sku] ?? ""}
                    onChange={(event) =>
                      setDraftQuantities((current) => ({
                        ...current,
                        [item.sku]: event.target.value,
                      }))
                    }
                  />
                  <div className="actions-row">
                    <button type="button" onClick={() => onUpdateQuantity(item.sku)}>
                      数量更新
                    </button>
                    <button type="button" onClick={() => onRemoveCartItem(item.sku)}>
                      商品削除
                    </button>
                  </div>
                </li>
              ))}
            </ul>
            {cartItems.length === 0 ? <p>カートは空です</p> : null}
            {cartMessage !== "" ? (
              <p role="status" className="status-message">
                {cartMessage}
              </p>
            ) : null}
          </section>

          <section className="subpanel">
            <h3>最終確認（US-CHK-03）</h3>
            <p>商品小計: {cartTotals.subtotal}</p>
            <p>送料: {cartTotals.shippingFee}</p>
            <p>手数料: {cartTotals.serviceFee}</p>
            <p>合計: {cartTotals.total}</p>
            <p>表示基準: サーバ算出値（mock）</p>
          </section>

          <section className="subpanel">
            <h3>注文履歴</h3>
            <button type="button" onClick={() => void onLoadHistory()}>
              履歴を取得
            </button>
            <ul>
              {historySummaries.map((summary) => (
                <li key={summary.orderId}>
                  {summary.orderId} / {summary.status} / {summary.createdAt} /{" "}
                  {summary.totalAmount}
                  <button type="button" onClick={() => void onLoadHistoryDetail(summary.orderId)}>
                    詳細を表示
                  </button>
                </li>
              ))}
            </ul>
            {historyDetail ? (
              <article className="result-card">
                <p>detail-orderId: {historyDetail.orderId}</p>
                <p>detail-status: {historyDetail.status}</p>
                <p>detail-customerId: {historyDetail.customerId}</p>
                <p>detail-subtotal: {historyDetail.subtotal}</p>
                <p>detail-shipping: {historyDetail.shippingFee}</p>
                <p>detail-serviceFee: {historyDetail.serviceFee}</p>
                <p>detail-total: {historyDetail.totalAmount}</p>
                <ul>
                  {historyDetail.items.map((item) => (
                    <li key={`${historyDetail.orderId}-${item.sku}`}>
                      {item.sku} x {item.quantity} ({item.lineAmount})
                    </li>
                  ))}
                </ul>
              </article>
            ) : null}
            {historyMessage !== "" ? (
              <p role="status" className="status-message">
                {historyMessage}
              </p>
            ) : null}
          </section>
        </>
      ) : null}
    </section>
  );
}
