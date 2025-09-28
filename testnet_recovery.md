# Chronos Testnet Recovery Guide

## 상황
- 테스트넷 validator node가 약 3일간 다운
- Finalized slot 2131360에서 checkpoint sync를 통해 복구 필요
- Single validator 환경으로 즉시 블록 생성 재개 필요

## 수정된 코드 요약

### 1. Initial Sync Service 수정 (`beacon-chain/sync/initial-sync/service.go`)
- Checkpoint recovery scenario 감지 로직 추가
- Slot 2131300-2131400 범위에서 checkpoint sync된 경우 즉시 synced로 마킹
- 대규모 slot gap에도 불구하고 block production 허용

### 2. Block Proposer 수정 (`beacon-chain/rpc/prysm/v1alpha1/validator/proposer.go`)
- Checkpoint recovery scenario에서 slot gap 제한 완화
- 확장된 timeout 설정 (최대 30분)
- State root 계산을 위한 더 긴 timeout (최대 45분)

## 복구 절차

### 1. 코드 빌드
```bash
cd /path/to/chronos
bazel build //cmd/beacon-chain:beacon-chain //cmd/validator:validator
```

### 2. 기존 데이터베이스 백업 (선택사항)
```bash
# 기존 beacon-chain 데이터 백업
mv ~/.eth2/beaconchaindata ~/.eth2/beaconchaindata.backup.$(date +%Y%m%d_%H%M%S)
```

### 3. Checkpoint Sync 실행
```bash
# Archive node URL에서 finalized checkpoint sync
./bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain \
  --checkpoint-sync-url=http://YOUR_ARCHIVE_NODE_URL \
  --min-sync-peers=0 \
  --execution-endpoint=http://localhost:8551 \
  --jwt-secret=path/to/jwt.hex \
  --verbosity=info \
  --accept-terms-of-use
```

### 4. Validator 시작 (별도 터미널)
```bash
./bazel-bin/cmd/validator/validator_/validator \
  --beacon-rpc-provider=localhost:4000 \
  --wallet-dir=path/to/wallet \
  --wallet-password-file=path/to/password.txt \
  --verbosity=info \
  --accept-terms-of-use
```

## 예상 동작

1. **Initial Sync 단계**:
   - Checkpoint sync를 통해 slot 2131360 상태 다운로드
   - 해당 slot 범위 감지시 즉시 "synced" 상태로 전환
   - 로그: "Checkpoint recovery detected - marking as synced to enable block production"

2. **Block Production 단계**:
   - 현재 slot과의 큰 gap에도 불구하고 block building 허용
   - 확장된 timeout으로 slot processing 진행
   - 로그: "Checkpoint recovery scenario detected - allowing block building with large gap"

3. **Gap Bridging**:
   - 한 번에 하나씩 slot을 처리하여 current slot까지 따라잡기
   - 각 slot당 최대 30분 timeout
   - 매우 큰 gap의 경우 최대 45분 timeout

## 모니터링 포인트

### 성공 지표
- 로그에서 "Checkpoint recovery detected" 메시지 확인
- Initial sync 완료 후 validator가 블록 제안 시작
- Slot gap이 점진적으로 줄어드는 것 확인

### 실패시 대응
1. **Sync 실패**: Archive node URL 확인 및 재시도
2. **Timeout 발생**: `--verbosity=debug`로 상세 로그 확인
3. **Execution client 연결 실패**: JWT secret 및 endpoint 확인

## 주의사항

1. **Single Validator 환경**: `--min-sync-peers=0` 설정 필수
2. **Archive Node**: Slot 2131360이 finalized된 상태여야 함
3. **Execution Client**: 해당 시점의 execution layer state 필요
4. **메모리 사용량**: 큰 slot gap 처리시 메모리 사용량 증가 예상

## 복구 후 확인사항

- [ ] Beacon chain head가 current slot에 근접
- [ ] Validator가 정상적으로 attestation 생성
- [ ] Block proposal이 성공적으로 수행
- [ ] P2P 연결 상태 정상
- [ ] Execution payload 생성 정상

## 트러블슈팅

### 로그 분석 키워드
- "Checkpoint recovery detected"
- "Single-validator setup detected with large slot gap"
- "Extended timeout for slot processing"
- "Marking as synced to enable block production"

### 성능 최적화
```bash
# 필요시 GC 설정 조정
export GOGC=50  # 더 빈번한 GC

# 메모리 제한 설정 (예: 16GB)
ulimit -v 16777216
```

## 완료 후 정리

1. 백업된 데이터베이스 정리 (복구 성공 확인 후)
2. 로그 파일 보관 (문제 분석용)
3. 모니터링 시스템 재설정