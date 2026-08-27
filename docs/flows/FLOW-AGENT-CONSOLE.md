<!-- toudocu
id: FLOW-AGENT-CONSOLE
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
updated: 2026-08-25
-->

# FLOW-AGENT-CONSOLE: Работа с coding agent

## Процесс

```mermaid
sequenceDiagram
    actor Human as Разработчик
    participant Task as Рабочая задача
    participant Console as Agent Console
    participant Provider as Codex app-server
    participant Sources as Changes, verification и feedback

    alt Новый запуск
        Human->>Task: Start work для Ready задачи
        Task->>Console: Передать актуальный контекст задачи
        Console->>Provider: Создать persistent thread и начать ответ
    else Продолжение запуска
        Human->>Console: Выбрать запуск из истории репозитория
        Console->>Provider: Проверить cwd и выполнить thread/resume
    end
    loop Пока сессия активна
        Provider-->>Console: Сообщения, команды, изменения и подтверждения
        Console-->>Human: Agent View и Command Output только для чтения
        opt Нужна корректировка активной работы
            Human->>Console: Дополнительное сообщение или Stop response
            Console->>Provider: Направить или прервать текущий ответ
        end
        opt Нужна операция Toudocu
            Human->>Console: Clarify, verification или Agent Feedback
            Console->>Sources: Выполнить существующее подготовленное действие
            Sources-->>Console: Вернуть фактический результат
        end
    end
    Human->>Console: Stop agent
    Console->>Provider: Завершить сессию
```

## Важные условия

- Agent Console не копирует состояние задачи, Changes, verification или Agent
  Feedback.
- Command Output коррелируется с карточкой команды по стабильному идентификатору
  и не является terminal emulator.
- Навигация меняет текущий контекст интерфейса, но не Agent Session.
- Историей threads и их транскриптами владеет Codex; Console только выводит
  список для текущего корня репозитория и запускает нативный resume.
- Approval, interrupt и stop не выводятся из текста команды.

## Связанные документы

- [UC-AGENT-CONSOLE-01](../use-cases/UC-AGENT-CONSOLE-01.md)
- [MOD-AGENT-CONSOLE](../modules/MOD-AGENT-CONSOLE.md)
- [контракт Agent Console](../contracts/agent-console.md)
