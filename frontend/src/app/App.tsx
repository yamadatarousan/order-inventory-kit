import { OrderCreateForm } from "../features/order-create/OrderCreateForm";
import { OrderActionsPanel } from "../features/order-actions/OrderActionsPanel";
import { MockLanePanels } from "../features/mock-lane/MockLanePanels";

export function App() {
  return (
    <main className="app-shell">
      <header className="app-hero">
        <p className="app-eyebrow">ORDER INVENTORY KIT</p>
        <h1>注文・在庫・決済コンソール</h1>
        <p className="app-lead">
          本実装レーンと仮実装レーンを同じ画面で確認できるようにした最小UIです。
        </p>
      </header>
      <div className="app-grid">
        <OrderCreateForm />
        <OrderActionsPanel />
        <MockLanePanels />
      </div>
    </main>
  );
}
