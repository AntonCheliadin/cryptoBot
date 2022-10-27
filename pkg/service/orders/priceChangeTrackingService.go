package orders

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
)

var serviceImpl *PriceChangeTrackingService

func NewPriceChangeTrackingService(priceChangeRepo repository.PriceChange) *PriceChangeTrackingService {
	if serviceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	serviceImpl = &PriceChangeTrackingService{
		priceChangeRepo: priceChangeRepo,
	}
	return serviceImpl
}

type PriceChangeTrackingService struct {
	priceChangeRepo repository.PriceChange
}

func (s *PriceChangeTrackingService) GetChangePrice(transactionId int64, currentPrice int64) *domains.PriceChange {
	priceChange, _ := s.priceChangeRepo.FindByTransactionId(transactionId)
	if priceChange != nil {
		s.saveNewPriceIfNeeded(priceChange, currentPrice)
	} else {
		priceChange = &domains.PriceChange{
			TransactionId: transactionId,
			LowPrice:      currentPrice,
			HighPrice:     currentPrice,
		}
		priceChange.RecalculatePercent()
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	}
	return priceChange
}

func (s *PriceChangeTrackingService) saveNewPriceIfNeeded(priceChange *domains.PriceChange, currentPrice int64) {
	if currentPrice > priceChange.HighPrice {
		priceChange.SetHigh(currentPrice)
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	} else if currentPrice < priceChange.LowPrice {
		priceChange.SetLow(currentPrice)
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	}
}
