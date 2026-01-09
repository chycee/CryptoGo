# 5. OS 표준 데이터 디렉토리 사용 (OS-Standard Data Directories)

날짜: 2026-01-07

## 현황 (Status)
제안됨 (Proposed)

## 맥락 (Context)
지금까지는 프로젝트 루트의 `_workspace/` 폴더에 모든 데이터를 저장했습니다. 이는 개발(Dev) 환경에서는 편리하지만, 컴파일된 바이너리를 시스템 경로에 설치하거나 배포할 때는 다음과 같은 문제가 있습니다:
1.  실행 권한 문제: 프로그램이 설치된 폴더(`C:\Program Files` 등)에 데이터 쓰기가 제한될 수 있음.
2.  사용자 격리: 여러 사용자가 PC를 쓸 경우 데이터가 섞일 위험이 있음.
3.  OS 관례 위반: 윈도우는 `AppData`, 리눅스는 `.local/share`를 사용하는 것이 표준입니다.

## 결정 (Decision)
**바이너리 실행 시 OS 표준 데이터 디렉토리를 최우선적으로 사용하도록 변경합니다.**

1.  **동적 경로 할당**:
    - **Windows**: `%AppData%/crypto-go`
    - **Linux**: `$XDG_DATA_HOME/crypto-go` 또는 `~/.local/share/crypto-go`
    - **macOS**: `~/Library/Application Support/crypto-go`
2.  **하위 호환성 (Fallback)**: 프로젝트 루트에 `_workspace/` 폴더가 이미 존재하는 경우(개발 환경)에는 해당 폴더를 우선 사용합니다.
3.  **설정 파일 검색**: `config.yaml`은 실행 파일 근처 -> 사용자 설정 폴더(`os.UserConfigDir`) 순서로 검색합니다.

## 결과 (Consequences)
-   **배포 용이성**: 바이너리 단독으로 배포하더라도 OS 표준에 맞춰 안전하게 데이터를 저장합니다.
-   **권한 안정성**: 사용자 홈 디렉토리 내에 저장하므로 관리자 권한 없이도 안정적으로 작동합니다.
-   **환경 분리**: 개발 환경과 실운용(Production) 환경이 명확히 분리됩니다.
