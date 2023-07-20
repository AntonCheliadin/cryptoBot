package orders

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
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

func (s *PriceChangeTrackingService) GetChangePrice(transactionId int64, currentPrice float64) *domains.PriceChange {
	priceChange, _ := s.priceChangeRepo.FindByTransactionId(transactionId)
	if priceChange != nil {
		s.saveNewPriceIfNeeded(priceChange, util.GetCents(currentPrice))
	} else {
		priceChange = &domains.PriceChange{
			TransactionId: transactionId,
			LowPrice:      util.GetCents(currentPrice),
			HighPrice:     util.GetCents(currentPrice),
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
