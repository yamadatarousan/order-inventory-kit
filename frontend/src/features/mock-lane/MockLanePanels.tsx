import { FormEvent, useMemo, useState } from "react";
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
  const [products, setProducts] = useState<MockProduct[]>([]);
  const [productDetailSKU, setProductDetailSKU] = useState("");
  const [productDetailMessage, setProductDetailMessage] = useState("");
  const [productDetail, setProductDetail] = useState<MockProduct | null>(null);

  const [cartSKU, setCartSKU] = useState("");
  const [cartQuantity, setCartQuantity] = useState("");
  const [cartMessage, setCartMessage] = useState("");
  const [cartItems, setCartItems] = useState<MockCartItem[]>([]);
  const [updateQuantities, setUpdateQuantities] = useState<Record<string, string>>({});

  const [historyCustomerID, setHistoryCustomerID] = useState("c-1");
  const [historyMessage, setHistoryMessage] = useState("");
  const [historySummaries, setHistorySummaries] = useState<MockOrderSummary[]>([]);
  const [historyDetail, setHistoryDetail] = useState<MockOrderDetail | null>(null);

  useState(() => {
    void listProducts().then((items) => setProducts(items));
  });

  const activeProducts = useMemo(
    () => products.filter((product) => product.isActive),
    [products],
  );
  const cartTotals = useMemo(() => calculateCartTotals(cartItems), [cartItems]);

  async function onLoadProductDetail(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProductDetailMessage("");
    setProductDetail(null);

    const sku = productDetailSKU.trim();
    if (sku === "") {
      setProductDetailMessage("入力内容を確認してください");
      return;
    }
    const result = await getProductDetail(sku);
    if (result.kind === "success") {
      setProductDetail(result.product);
      return;
    }
    setProductDetailMessage("商品が見つかりません (404)");
  }

  function onAddCartItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCartMessage("");

    const sku = cartSKU.trim();
    const quantity = Number(cartQuantity);
    if (sku === "" || !Number.isInteger(quantity) || quantity < 1) {
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
    const raw = updateQuantities[sku] ?? "";
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

  async function onLoadHistory(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setHistoryMessage("");
    setHistoryDetail(null);

    const customerId = historyCustomerID.trim();
    if (customerId === "") {
      setHistoryMessage("入力内容を確認してください");
      return;
    }

    const summaries = await listHistory(customerId);
    setHistorySummaries(summaries);
  }

  async function onLoadHistoryDetail(orderId: string) {
    const customerId = historyCustomerID.trim();
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
    <section>
      <h2>仮実装レーン（mock）</h2>

      <section>
        <h3>商品一覧</h3>
        <p>販売中の商品のみを表示します。</p>
        <ul>
          {activeProducts.map((product) => (
            <li key={product.sku}>
              {product.sku} / {product.name} / unit_price:{product.unitPrice} /{" "}
              {product.currency} / is_active:{String(product.isActive)}
              <button
                type="button"
                onClick={() => {
                  setCartSKU(product.sku);
                }}
              >
                カート追加候補に設定
              </button>
            </li>
          ))}
        </ul>

        <form onSubmit={onLoadProductDetail}>
          <label htmlFor="productDetailSKU">商品詳細SKU</label>
          <input
            id="productDetailSKU"
            name="productDetailSKU"
            value={productDetailSKU}
            onChange={(event) => setProductDetailSKU(event.target.value)}
          />
          <button type="submit">商品詳細を表示</button>
        </form>
        {productDetail ? (
          <p>
            detail: {productDetail.sku} / {productDetail.name} /{" "}
            {productDetail.unitPrice}
          </p>
        ) : null}
        {productDetailMessage !== "" ? <p role="status">{productDetailMessage}</p> : null}
      </section>

      <section>
        <h3>カート操作</h3>
        <form onSubmit={onAddCartItem}>
          <label htmlFor="cartSKU">カートSKU</label>
          <input
            id="cartSKU"
            name="cartSKU"
            value={cartSKU}
            onChange={(event) => setCartSKU(event.target.value)}
          />
          <label htmlFor="cartQuantity">カート数量</label>
          <input
            id="cartQuantity"
            name="cartQuantity"
            type="number"
            inputMode="numeric"
            value={cartQuantity}
            onChange={(event) => setCartQuantity(event.target.value)}
          />
          <button type="submit">カートに追加</button>
        </form>

        <ul>
          {cartItems.map((item) => (
            <li key={item.sku}>
              {item.sku} x {item.quantity} = {item.lineAmount}
              <label htmlFor={`qty-${item.sku}`}>更新数量</label>
              <input
                id={`qty-${item.sku}`}
                value={updateQuantities[item.sku] ?? ""}
                onChange={(event) =>
                  setUpdateQuantities((current) => ({
                    ...current,
                    [item.sku]: event.target.value,
                  }))
                }
              />
              <button type="button" onClick={() => onUpdateQuantity(item.sku)}>
                数量更新
              </button>
              <button type="button" onClick={() => onRemoveCartItem(item.sku)}>
                商品削除
              </button>
            </li>
          ))}
        </ul>
        <p>subtotal: {cartTotals.subtotal}</p>
        <p>shipping: {cartTotals.shippingFee}</p>
        <p>serviceFee: {cartTotals.serviceFee}</p>
        <p>total: {cartTotals.total} (server-calculated mock)</p>
        {cartMessage !== "" ? <p role="status">{cartMessage}</p> : null}
      </section>

      <section>
        <h3>最終確認（US-CHK-03）</h3>
        <p>商品小計: {cartTotals.subtotal}</p>
        <p>送料: {cartTotals.shippingFee}</p>
        <p>手数料: {cartTotals.serviceFee}</p>
        <p>合計: {cartTotals.total}</p>
        <p>表示基準: サーバ算出値（mock）</p>
      </section>

      <section>
        <h3>注文履歴</h3>
        <form onSubmit={onLoadHistory}>
          <label htmlFor="historyCustomerId">履歴customerId</label>
          <input
            id="historyCustomerId"
            name="historyCustomerId"
            value={historyCustomerID}
            onChange={(event) => setHistoryCustomerID(event.target.value)}
          />
          <button type="submit">履歴を取得</button>
        </form>
        <ul>
          {historySummaries.map((summary) => (
            <li key={summary.orderId}>
              {summary.orderId} / {summary.status} / {summary.createdAt} /{" "}
              {summary.totalAmount}
              <button type="button" onClick={() => onLoadHistoryDetail(summary.orderId)}>
                詳細を表示
              </button>
            </li>
          ))}
        </ul>
        {historyDetail ? (
          <article>
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
        {historyMessage !== "" ? <p role="status">{historyMessage}</p> : null}
      </section>
    </section>
  );
}
