package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"order-service/internal/models"
	"time"

	_ "github.com/lib/pq"
)

type Postgres struct {
	db *sql.DB
}

func New(connStr string) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Postgres{db: db}, nil
}

func (p *Postgres) Close() error {
	return p.db.Close()
}

func (p *Postgres) SaveOrder(order *models.Order) error {
	ctx := context.Background()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (order_uid, track_number, "entry", locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO deliveries (order_uid, delivery_name, phone, zip, city, delivery_address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (order_uid, payment_transaction, request_id, currency, payment_provider, 
			amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM items WHERE order_uid = $1", order.OrderUID)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, item_status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Postgres) GetOrder(orderUID string) (*models.Order, error) {
	log.Printf("Getting order %s from database", orderUID)

	ctx := context.Background()

	var order models.Order
	err := p.db.QueryRowContext(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1`, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
	)
	if err != nil {
		return nil, err
	}

	err = p.db.QueryRowContext(ctx, `
		SELECT delivery_name, phone, zip, city, delivery_address, region, email
		FROM deliveries WHERE order_uid = $1`, orderUID).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
	)
	if err != nil {
		log.Printf("Error getting order %s: %v", orderUID, err)

		return nil, err
	}

	err = p.db.QueryRowContext(ctx, `
		SELECT payment_transaction, request_id, currency, payment_provider, amount, payment_dt, 
			bank, delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid = $1`, orderUID).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx, `
		SELECT chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, item_status
		FROM items WHERE order_uid = $1`, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale,
			&item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	order.Items = items

	return &order, nil
}

func (p *Postgres) GetAllOrders() (map[string]*models.Order, error) {
    log.Printf("Getting all orders from database...")
    
    ctx := context.Background()
    rows, err := p.db.QueryContext(ctx, "SELECT order_uid FROM orders")
    if err != nil {
        log.Printf("Query error: %v", err)
        return nil, err
    }
    defer rows.Close()

    orders := make(map[string]*models.Order)
    count := 0
    
    for rows.Next() {
        var orderUID string
        if err := rows.Scan(&orderUID); err != nil {
            log.Printf("Error scanning order UID: %v", err)
            continue
        }
        
        log.Printf("Found order: %s", orderUID)
        order, err := p.GetOrder(orderUID)
        if err != nil {
            log.Printf("Error getting order %s: %v", orderUID, err)
            continue
        }
        
        orders[orderUID] = order
        count++
    }
    
    log.Printf("Loaded %d orders from database", count)
    return orders, nil
}

func (p *Postgres) CreateOrderFromJSON(jsonData []byte) (*models.Order, error) {
    var order models.Order
    if err := json.Unmarshal(jsonData, &order); err != nil {
        return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
    }

    if order.DateCreated.IsZero() {
        var raw map[string]interface{}
        if err := json.Unmarshal(jsonData, &raw); err == nil {
            if dateStr, ok := raw["date_created"].(string); ok {
                if parsedTime, err := time.Parse(time.RFC3339, dateStr); err == nil {
                    order.DateCreated = parsedTime
                }
            }
        }
    }

    return &order, nil
}