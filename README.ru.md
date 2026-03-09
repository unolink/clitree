[English version (README.md)](README.md)

# clitree

Легковесный декларативный CLI-фреймворк для Go без зависимостей.

## Возможности

- **Декларативный** — команды определяются как структуры данных, а не цепочки методов
- **Вложенные подкоманды** — неограниченная глубина с автоматическим разрешением путей
- **Автогенерация help** — форматируется из метаданных команд
- **Алиасы** — альтернативные имена команд (например, `-v` для `version`)
- **Скрытые команды** — deprecated-команды невидимы в help, но исполняемы
- **Тестируемый** — подставляемые stdout/stderr writers
- **Без зависимостей** — только стандартная библиотека Go

## Установка

```bash
go get github.com/unolink/clitree
```

## Использование

```go
root := &clitree.Command{
    Name:  "myapp",
    Short: "My CLI application",
    Subcommands: []*clitree.Command{
        {
            Name:    "version",
            Aliases: []string{"-v", "--version"},
            Short:   "Show version",
            Run: func(ctx context.Context, fs *flag.FlagSet) error {
                fmt.Println("v1.0.0")
                return nil
            },
        },
        {
            Name:  "serve",
            Short: "Start server",
            Flags: func(fs *flag.FlagSet) {
                fs.String("addr", ":8080", "Listen address")
            },
            Run: func(ctx context.Context, fs *flag.FlagSet) error {
                addr := fs.Lookup("addr").Value.String()
                fmt.Printf("Listening on %s\n", addr)
                return nil
            },
        },
    },
}

tree := clitree.New(root, os.Stdout, os.Stderr)
os.Exit(tree.Execute(context.Background(), os.Args[1:]))
```

## Сравнение с Cobra

| Возможность            | clitree | cobra |
|------------------------|---------|-------|
| Подкоманды             | да      | да    |
| Авто-help              | да      | да    |
| Алиасы                 | да      | да    |
| Парсинг флагов         | да      | да    |
| Зависимости            | 0       | 4+    |
| Оверхед бинарника      | ~0 KB   | ~2 MB |

## Лицензия

MIT — см. [LICENSE](LICENSE).
