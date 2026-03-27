# Onboarding Foundation Notes

## What This Commit Establishes

This plugin foundation implements the first practical layer of the onboarding automation described in the requirements draft:

- Automatic onboarding trigger on `UserHasBeenCreated`
- Bot-authored DM delivery using Mattermost plugin APIs
- Common and department template composition
- Link activation and sort order handling
- Variable replacement for user and department fields
- Fallback to common-only or fallback department template
- Exclusion policy evaluation
- KV-backed send logs
- Automatic-send duplicate prevention
- Retry queue with scheduled processing
- Admin-only preview, resend, stats, and log APIs
- System Console configuration for templates, links, mappings, and exclusions

## Requirement Coverage

Covered directly in this foundation:

- `FR-01` 신규 사용자 자동 감지
- `FR-02` 자동 DM 전송
- `FR-03` 공통 메시지 지원
- `FR-04` 부서별 메시지 지원
- `FR-05` 조합형 메시지
- `FR-06` 기본 fallback
- `FR-07` 템플릿 변수 치환
- `FR-08` 링크 목록 관리
- `FR-10` 최초 1회 발송 제어 및 수동 재전송 API
- `FR-11` 대상 제외 정책
- `FR-12` 미리보기 API
- `FR-15` 최소 공통 메시지 fallback
- `FR-16` 발송 로그
- `FR-18` 언어 코드 기반 템플릿 구조
- `NFR-02` 실패 시 재시도 큐 기반 처리
- `NFR-03` 중복 방지
- `NFR-07` 코드 배포 없는 운영 설정 변경
- `NFR-08` 기본 운영 통계 API

Partially covered and intended as next steps:

- `FR-09` 템플릿 버전 이력은 JSON 내 `version` 필드까지 반영, 변경 이력 저장은 미구현
- `FR-13` 전용 Admin UI는 미구현, 현재는 System Console 설정과 Admin API 사용
- `FR-14` HR 연동은 미구현, 현재는 Mattermost 프로필 props + 매핑 JSON 기반
- `FR-17` 운영 통계는 KV 로그 집계 기반의 기본 API만 제공
- `NFR-01` 1분 이내 발송은 초기 지연 및 재시도 정책에 따라 충족되도록 운영 설정 필요
- `NFR-05` 감사 추적은 발송 로그만 우선 구현, 템플릿 변경 감사는 후속 과제

## Suggested Next Steps

1. Replace JSON blob management with a dedicated admin web UI backed by plugin APIs.
2. Introduce a persistent template repository abstraction so KV can be swapped for SQL without changing delivery logic.
3. Add HR adapter integration for department, organization, hire date, and employment status.
4. Persist configuration change audit history with editor identity and timestamp.
5. Extend the queue to support day-1, day-7, and day-30 phased onboarding journeys.
