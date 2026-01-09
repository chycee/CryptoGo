# 1. 아키텍처 결정 기록(ADR) 도입

날짜: 2026-01-07

## 현황 (Status)
승인됨 (Accepted)

## 맥락 (Context)
프로젝트가 진행됨에 따라 중요한 아키텍처 결정사항(예: 구조적 보안, 폴더 구조 등)이 발생하고 있습니다.
단순한 커밋 메시지나 채팅 로그만으로는 의사결정의 "맥락(Context)"과 "이유(Why)"를 보존관리(Version Control)하기 어렵습니다.
현업(Industry Standard)에서는 생각의 흐름(Sequential Thinking)을 체계적으로 관리하기 위해 **ADR (Architecture Decision Records)**을 사용합니다.

## 결정 (Decision)
우리는 프로젝트의 중요 의사결정을 `docs/adr/` 디렉토리 하위에 번호가 매겨진 Markdown 파일로 기록합니다.
형식은 [Michael Nygard의 템플릿](https://github.com/joelparkerhenderson/architecture-decision-record)을 따르며, 한국어로 작성합니다.

## 결과 (Consequences)
-   의사결정의 이력이 영구적으로 보존됩니다.
-   새로운 팀원(또는 미래의 자신)이 "왜 이렇게 만들었는가?"를 이해하기 쉬워집니다.
-   문서화 오버헤드가 약간 발생하지만, "Quant"의 유지보수성을 위해 필수적입니다.
