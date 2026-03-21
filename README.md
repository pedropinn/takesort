# TakeSort

Uma maneira facil de organizar suas filmagens. Copie tudo para uma pasta e o TakeSort se encarrega de organizar os arquivos para voce.

Ele monitora a pasta `/temp` e move cada arquivo para `/media/{ano}/{ano-mes-dia}/` conforme o tipo:

| Extensao | Destino |
|----------|---------|
| `.mp4` | `{data}/` |
| `.jpg` | `{data}/photos/jpeg/` |
| `.dng` | `{data}/photos/raw/` |
| `.wav` | `{data}/audio/` |
| `.lrf`, `.lrv` | `{data}/proxy/` (renomeado para .mp4) |
| `.thm`, `.srt` | Deletado |

## Subir local

```bash
docker compose up -d
```

Os arquivos vao para `./data/temp` (entrada) e `./data/media` (saida organizada).

Para ver os logs:

```bash
docker compose logs -f
```

## Configuracao

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `TAKESORT_DEBOUNCE_INTERVAL` | `2s` | Tempo entre verificacoes de estabilidade do arquivo |
| `TAKESORT_LOG_LEVEL` | `info` | Nivel de log (debug, info, warn, error) |
