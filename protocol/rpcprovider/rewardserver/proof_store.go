package rewardserver

import (
	"context"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ProofEntity struct {
	epoch     int64
	consumer  string
	sessionId uint64
	proof     *pairingtypes.RelaySession
}

type proofEntityInternal struct {
	epoch     int64
	consumer  string
	sessionId uint64
	proof     []byte
}

type ProofStore struct {
	store []*proofEntityInternal
}

func (s *ProofStore) Save(ctx context.Context, consumer string, proof *pairingtypes.RelaySession) error {
	proofData, err := proof.Marshal()
	if err != nil {
		return err
	}

	s.store = append(s.store, &proofEntityInternal{
		epoch:     proof.Epoch,
		consumer:  consumer,
		sessionId: proof.SessionId,
		proof:     proofData,
	})

	return nil
}

func (s *ProofStore) FindAll(ctx context.Context) ([]ProofEntity, error) {
	var result []ProofEntity

	for _, proofEntityInternal := range s.store {
		proof := &pairingtypes.RelaySession{}
		err := proof.Unmarshal(proofEntityInternal.proof)
		if err != nil {
			return nil, err
		}

		result = append(result, ProofEntity{
			epoch:     proofEntityInternal.epoch,
			consumer:  proofEntityInternal.consumer,
			sessionId: proof.SessionId,
			proof:     proof,
		})
	}

	return result, nil
}

func NewProofStore() *ProofStore {
	return &ProofStore{
		store: []*proofEntityInternal{},
	}
}
