package stateutil

import (
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestValidatorConstants(t *testing.T) {
	v := &ethpb.Validator{}
	refV := reflect.ValueOf(v).Elem()
	numFields := refV.NumField()
	numOfValFields := 0

	for i := 0; i < numFields; i++ {
		if strings.Contains(refV.Type().Field(i).Name, "state") ||
			strings.Contains(refV.Type().Field(i).Name, "sizeCache") ||
			strings.Contains(refV.Type().Field(i).Name, "unknownFields") {
			continue
		}
		numOfValFields++
	}
	assert.Equal(t, validatorFieldRoots, numOfValFields)
	expectedTreeDepth := math.Ceil(math.Log2(float64(validatorFieldRoots)))
	assert.Equal(t, validatorTreeDepth, int(expectedTreeDepth))

	_, err := ValidatorRegistryRoot([]*ethpb.Validator{v})
	assert.NoError(t, err)
}

func TestHashValidatorHelper(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	v := &ethpb.Validator{}
	valList := make([]*ethpb.Validator, 10*validatorFieldRoots)
	for i := range valList {
		valList[i] = v
	}
	roots := make([][32]byte, len(valList))
	hashValidatorHelper(valList, roots, 2, 2, &wg)
	for i := 0; i < 4*validatorFieldRoots; i++ {
		require.Equal(t, [32]byte{}, roots[i])
	}
	emptyValRoots, err := ValidatorFieldRoots(v)
	require.NoError(t, err)
	for i := 4; i < 6; i++ {
		for j := 0; j < validatorFieldRoots; j++ {
			require.Equal(t, emptyValRoots[j], roots[i*validatorFieldRoots+j])
		}
	}
	for i := 6 * validatorFieldRoots; i < 10*validatorFieldRoots; i++ {
		require.Equal(t, [32]byte{}, roots[i])
	}
}

func BenchmarkTestValidatorRegistryRoot(b *testing.B) {
	valList := make([]*ethpb.Validator, 1_000_000)
	for i := range valList {
		valList[i] = &ethpb.Validator{
			PublicKey:        []byte{byte(i)},
			EffectiveBalance: uint64(i),
			PrincipalBalance: uint64(i),
		}
	}
	b.Run("100k validators", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := validatorRegistryRoot(valList)
			require.NoError(b, err)
		}
	})
}
