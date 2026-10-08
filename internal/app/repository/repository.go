package repository

import "sync"

// Repository — in-memory хранилище услуг, потокобезопасное.
type Repository struct {
	mu     sync.RWMutex
	orders []Order
	nextID int
}

func New() *Repository {
	return &Repository{nextID: 1}
}

// Seed наполняет репозиторий начальными данными.
func (r *Repository) Seed(orders []Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range orders {
		o.ID = r.nextID
		r.nextID++
		r.orders = append(r.orders, o)
	}
}

// Published возвращает все опубликованные услуги.
func (r *Repository) Published() []Order {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Order, 0, len(r.orders))
	for _, o := range r.orders {
		if o.Status == StatusPublished {
			out = append(out, o)
		}
	}
	return out
}

// Draft возвращает единственный черновик (если есть).
func (r *Repository) Draft() (Order, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orders {
		if o.Status == StatusDraft {
			return o, true
		}
	}
	return Order{}, false
}

// ByID ищет услугу по ID, исключая deleted.
func (r *Repository) ByID(id int) (Order, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orders {
		if o.ID == id && o.Status != StatusDeleted {
			return o, true
		}
	}
	return Order{}, false
}

// NextPublished возвращает следующую опубликованную услугу после id (с зацикливанием).
func (r *Repository) NextPublished(id int) (Order, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pub := make([]Order, 0, len(r.orders))
	for _, o := range r.orders {
		if o.Status == StatusPublished {
			pub = append(pub, o)
		}
	}
	if len(pub) == 0 {
		return Order{}, false
	}
	for i, o := range pub {
		if o.ID == id {
			return pub[(i+1)%len(pub)], true
		}
	}
	// Если id не найден — возвращаем первый.
	return pub[0], true
}

// FilterByPrice возвращает опубликованные услуги с ценой в диапазоне [min; max].
// min=0 или max=0 означает «без ограничения» с соответствующей стороны.
func (r *Repository) FilterByPrice(min, max float64) []Order {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Order, 0, len(r.orders))
	for _, o := range r.orders {
		if o.Status != StatusPublished {
			continue
		}
		if min > 0 && o.Price < min {
			continue
		}
		if max > 0 && o.Price > max {
			continue
		}
		out = append(out, o)
	}
	return out
}
