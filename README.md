# TakeSort

Uma maneira facil de organizar suas filmagens. Copie tudo para uma pasta e o TakeSort se encarrega de organizar os arquivos para voce.

Ele monitora a pasta de entrada e move cada arquivo para `{TAKESORT_MEDIA_DIR}/{ano}/{ano-mes-dia}/` conforme o tipo:

| Extensao | Destino |
|----------|---------|
| `.mp4` | `{data}/` |
| `.jpg` | `{data}/photos/jpeg/` |
| `.dng` | `{data}/photos/raw/` |
| `.wav` | `{data}/audio/` |
| `.lrf`, `.lrv` | `{data}/proxy/` |
| `.thm`, `.srt` | Deletado |

## Docker Compose
```yaml
services:
  takesort:
    image: pedropinn/takesort:latest
    user: "3000:3000"
    volumes:
      - /mnt/hdd/media/videos:/media
    environment:
      - TAKESORT_DEBOUNCE_INTERVAL=2s
      - TAKESORT_LOG_LEVEL=info
    restart: unless-stopped
```

## Configuracao

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `TAKESORT_WATCH_DIR` | `/media/temp` | Pasta monitorada para novos arquivos |
| `TAKESORT_MEDIA_DIR` | `/media` | Pasta raiz para arquivos organizados |
| `TAKESORT_DEBOUNCE_INTERVAL` | `2s` | Tempo entre verificacoes de estabilidade do arquivo |
| `TAKESORT_LOG_LEVEL` | `info` | Nivel de log (debug, info, warn, error) |

Para melhor performance, monte watch e media no mesmo volume/filesystem para que `os.Rename` funcione sem copiar bytes.
