-- seedで投入した固定SKUのみ削除する。
DELETE FROM inventory
WHERE sku IN ('sku-1', 'sku-2', 'sku-3');
