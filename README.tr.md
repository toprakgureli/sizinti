🇹🇷 Türkçe | 🇬🇧 [English](README.md)

# sizinti

[![CI iş akışı](https://img.shields.io/badge/CI-workflow%20included-blue)](.github/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8)
[![Lisans: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Go ile yazılmış, eşzamanlı çalışan bir tarayıcı. Koda kaçan kimlik bilgilerini (API anahtarı, parola, token) commit'e girmeden yakalar; hem global hem de Türk sağlayıcılar için kurallarla gelir.

## Neden geliştirdim?

Backend mühendisi ve eski bir sızma testçisi olarak kimlik bilgilerinin henüz geliştiricinin makinesindeyken yakalanmasını önemsiyorum. Sizinti'yi bu sorunu açıklayabildiğim, okunabilir Go koduyla ele almak ve Türk sağlayıcı yapılandırmalarına ayrıca odaklanmak için geliştirdim.

Sizinti yerel dosyaları okuyup olası kimlik bilgilerini raporlar. Bulduğu değerleri bir sağlayıcıya bağlanmak için kullanmaz. CI rozeti mevcut iş akışına bağlantıdır; projeyi yayımlayıp CI'ı çalıştırdıktan sonra canlı durum rozetiyle değiştirebilirsin.

## Ne zaman kullanılır?

- **Commit öncesinde:** `sizinti .` komutunu yerelde veya mevcut bir pre-commit hook içinden çalıştır.
- **Push veya pull request sırasında:** tespit edilen kimlik bilgilerinin ilerlemesini engellemek için zorunlu CI kontrolü olarak kullan.
- **Repo denetiminde:** takip edilmeyen dosyaları ve gerektiğinde fixture'ları da içeren çalışma ağacını tara.

Mevcut tarama diskteki dosyaları kapsar. Stage edilmiş sürüm çalışma ağacından farklı olabilir; staged index ve Git geçmişi taraması yol haritasında.

## Özellikler

- Sınırlı worker pool, context ile iptal ve deterministik sonuç sırası.
- Regex kuralları ve eşleşmeyen string'ler için Shannon entropy tespiti.
- AWS, GCP bağlamı, GitHub, Slack, Stripe, JWT, PEM, genel atama ve veritabanı URL kuralları.
- Yaklaşık iyzico, PayTR, Param, Netgsm ve Türk banka/API atama kuralları.
- Dosya yolu, satır, kural ve güven bilgisiyle tamamen maskelenmiş bulgular.
- Renkli terminal tablosu, JSON ve SARIF 2.1.0 çıktısı.
- Glob ve regex allowlist, isteğe bağlı YAML yapılandırması ve açık bayrak önceliği.
- İkili dosya, boyut, bağımlılık ve yaygın fixture istisnaları.

## Kurulum

Go 1.25 veya üzeri gerekir. Kaynak kod dizininde:

```sh
go mod download
go build -o bin/sizinti ./cmd/sizinti
go install ./cmd/sizinti
```

Windows'ta `go build -o bin/sizinti.exe ./cmd/sizinti` kullan. `go install`, çalıştırılabilir dosyayı `GOBIN` dizinine; bu değişken tanımlı değilse `GOPATH/bin` dizinine yazar. İlgili dizini PATH'e ekle.

Modül yolu `github.com/toprakgureli/sizinti`.

## Hızlı başlangıç

```sh
sizinti .
sizinti ../backend --workers 4 --entropy-threshold 4.8
sizinti . --json > ../sizinti-result.json
sizinti . --sarif > ../sizinti-result.sarif
sizinti . --include-fixtures --max-file-size 20971520
sizinti . --config sizinti.example.yaml
```

Tarayıcının kendi çıktısını okumaması için raporları taranan dizinin dışına yaz.

| Çıkış kodu | Anlamı |
| --- | --- |
| 0 | Taranan kapsamda bulgu yok |
| 1 | Olası kimlik bilgileri bulundu |
| 2 | Geçersiz girdi, iptal, okuma veya çıktı hatası |

Hatalar stderr'e, JSON/SARIF stdout'a yazılır. Tarama hatasında rapor üretilmez; boş dosyayı yorumlamadan önce çıkış kodunu kontrol et. Çıkış kodlarını test ederken `go run` yerine derlenmiş programı kullan; çünkü `go run` programın kendi çıkış kodunu tam yansıtmaz.

## Örnek çıktı

Projeyle gelen fixture, sentetik `iyzico_api_key="synthetic-only"` atamasını içerir.

```sh
sizinti testdata/synthetic --no-color
```

```text
FILE       LINE  RULE    CONFIDENCE  SNIPPET
"app.env"  1     iyzico  medium      [REDACTED]
1 finding(s); 1 scanned; 0 skipped entries
```

Bu komut `1` koduyla çıkar. Raporlarda kimlik bilgisi veya kaynak satırı yerine `[REDACTED]` bulunur. Dosya yolları görünür kalır.

## Nasıl çalışır?

Producer dizini gezip yolları tamponsuz bir kanala gönderir. Worker'lar sınırlı tamponla dosyanın ikili içerik taşıyıp taşımadığını kontrol eder, başa döner ve satır satır tarar. Bulgular tek collector'a aittir; sonuçlar dosya yolu, satır ve kurala göre sıralanır.

Özel regex kuralları, Türk sağlayıcı atamalarından ve genel atamalardan önce çalışır. Çakışan eşleşmeler bir kez raporlanır. Eşleşmeyen, en az 20 baytlık token benzeri string'ler entropy eşiğiyle karşılaştırılır ve düşük güven etiketi alır.

Her satır, ayırıcı payı dahil 1 MiB scanner tamponuna sığmalıdır; daha uzun satırlar hata üretir. Dosya okuma tamponları sınırlıdır; bulgular ve dizin kayıtları ayrıca bellek kullanır. [DESIGN.md](DESIGN.md), sahiplik ve concurrency modelini açıklar.

## Yapılandırma

Ayarlar şu sırayla uygulanır: önce varsayılanlar, sonra `--config` ile verdiğin YAML dosyası, en son da komut satırında açıkça verdiğin bayraklar. YAML dosyası en fazla 1 MiB olabilir; bilinmeyen ya da tekrar eden alanlar ve birden fazla belge hata verir.

| Ayar / bayrak | Varsayılan | Anlamı |
| --- | --- | --- |
| `workers` / `--workers` | `runtime.NumCPU()` | Worker sayısı, 1–1024 |
| `entropy-threshold` / `--entropy-threshold` | `4.5` | Sonlu bit/bayt eşiği, 0–8 |
| `max-file-size` / `--max-file-size` | `10485760` | Bayt cinsinden dosya boyutu sınırı |
| `include-fixtures` / `--include-fixtures` | `false` | Yaygın test/örnek yollarını dahil et |
| `ignore-file` / `--ignore-file` | `.sizintiignore` | Tarama köküne göreli veya mutlak yol |
| `format` | `table` | YAML: `table`, `json` veya `sarif` |
| `no-color` / `--no-color` | `false` | Tablo rengini kapat |

`--json` ve `--sarif`, yapılandırılmış biçimi değiştirir ve birlikte kullanılamaz. `--include-fixtures=false` gibi açık Boolean bayrakları YAML'ı geçersiz kılar. `--ignore-file=` ignore yüklemeyi kapatır. Varsayılan `.sizintiignore` bulunmayabilir; farklı isimle belirtilen dosyanın bulunmaması hatadır. Renk için karakter aygıtı çıktısı gerekir; `NO_COLOR` veya `--no-color` rengi kapatır.

Config yolu geçerli çalışma dizinine görelidir. Ignore dosyasının göreli yolu ve glob'lar tarama kökünden çözülür.

## False positive ve istisnalar

Örnek `.sizintiignore`:

```text
glob:go.sum
glob:*.sample
glob:config/*.local
regex:synthetic-[a-z]+
```

Boş satırlar ve `#` ile başlayan satırlar yok sayılır. Glob'lar Go `path.Match` sözdizimini kullanır: `*`, `?` ve karakter sınıfları; her işletim sisteminde `/` kullanılır. `/` içermeyen desenler her derinlikte dosya/dizin adını, `/` içerenler köke göreli yolu eşleştirir. Bir dizinin eşleşmesi alt ağacını dışarıda bırakır. Özyinelemeli `**`, olumsuzlama ve Gitignore semantiği desteklenmez.

Regex istisnaları adayın tamamıyla eşleşir; bir değere izin vermek aynı satırdaki diğer kimlik bilgilerini kapsam dışına çıkarmaz. İstisnaları dar ve sentetik tut.

Varsayılan istisnalar:

- `.git`, `vendor` ve `node_modules` dizinleri.
- `testdata`, `fixtures` ve `examples` dizinleri; `*_test.go`, `*.example` ve `*.sample` dosyaları. Dahil etmek için `--include-fixtures` kullan.
- Sembolik bağlantılar, normal dosya olmayan girdiler ve boyut sınırını aşan dosyalar.
- 9'dan küçük veya 14–31 aralığındaki kontrol baytlarını içeren dosyalar. Bu ikili içerik sezgiseli, NUL içeren UTF-16 dosyalarını da dışlar.

Açıkça seçilen kök, adı `fixtures` veya `testdata` olsa da taranır. Atlanan her dizin tek bir atlanmış girdi sayılır. `.gitignore` yüklenmez; takip edilmeyen `.env` dosyaları taramaya uygundur. Denetimlerde `--include-fixtures` ile birlikte istisnaları ve boyut sınırlarını gözden geçir.

Yüksek entropy eşikleri gürültüyü azaltır ama daha fazla kimlik bilgisini kaçırır; düşük eşikler duyarlılığı ve false positive sayısını artırır. Hash'ler ve rastgele kimlikler erişim sağlamadan da yüksek entropy taşıyabilir. Genel atamalar zararsız ifadelerle de eşleşebilir. Dört bayttan kısa değerler ve tanınan işaret içermeyen çok satırlı değerler kaçabilir.

## Kurallar ve güven

| Kategori | Tanıma yöntemi | Güven |
| --- | --- | --- |
| AWS | `AKIA`/`ASIA` kimlikleri; `aws_secret_access_key` yanındaki 40 karakterlik değerler | Yüksek |
| GCP | JSON `type: service_account`; PEM üzerinden özel anahtarlar | Bağlam için düşük, PEM için yüksek |
| GitHub | `ghp_` / `gho_` ve ardından 36 alfanümerik karakter | Yüksek |
| Slack / Stripe | Slack token prefiksleri; `sk_live_` / `sk_test_` biçimleri | Yüksek |
| JWT | `eyJ` başlığıyla başlayan, token benzeri üç parça | Orta |
| PEM | RSA, EC, DSA, OpenSSH ve şifreli türler dahil özel anahtar BEGIN işaretleri | Yüksek |
| Veritabanı URL'leri | Parola içeren PostgreSQL, MySQL, MongoDB URL'leri | Yüksek |
| Genel | `api_key`, `secret`, `password`, `passwd`, `token`, `client_secret` atamaları | Orta |
| Türk sağlayıcılar | Sağlayıcı prefiksli atamalar | Orta |
| Entropy | Eşleşmeyen, en az 20 baytlık token benzeri string'ler | Düşük |

Güven etiketi kimlik bilgisinin çalıştığını değil, desenini ifade eder. GCP işareti özel anahtar olmadan da bulunabilir; PEM tespiti bloğu doğrulamak yerine başlangıç işaretini tanır; JWT'ler biçim üzerinden eşleştirilir.

Türk kuralları yaklaşık ve katkıya açıktır. Kaynakta tanımlanan alt çizgi/tire varyasyonlarıyla, büyük-küçük harf duyarsız alan adlarını eşleştirir:

- iyzico: `iyzico_api_key`, `iyzico_secret_key`.
- PayTR: `paytr_merchant_key`, `paytr_merchant_salt`, `paytr_api_key`.
- Param: `param_client_code`, `param_client_username`, `param_client_password`, `param_guid`, `param_api_key`.
- Netgsm: `netgsm_password`, `netgsm_usercode`, `netgsm_api_key`.
- Banka/API: `banka`, `bank`, `turkish_bank`, `tr_bank` prefiksleriyle `api_key`, `client_secret` veya `password`.

Bazı kimlikler herkese açık olabilir. Farklı adlandırmalar sağlayıcıya özel etiketin kaybolmasına da yol açabilir. Katkılar sağlayıcı dokümanı ve sentetik pozitif/negatif örnekler içermeli; çalışan kimlik bilgileri içermemeli.

## CI ve hook'lar

`sizinti .` komutunu çıkış kodunu koruyarak zorunlu pipeline kontrolü veya mevcut pre-commit hook içinde çalıştır. GitHub SARIF yüklemeleri için repo kökünden tara ve raporu dışarıya yaz. `github/codeql-action/upload-sarif` ve `security-events: write` kullan; `1` çıkışından sonra taramanın başarısız durumunu koruyarak yüklemeye izin ver, `2` çıkışında yükleme yapma. Yüklemeyi sizinti değil CI gerçekleştirir. Ayrıntılar: [GitHub SARIF desteği](https://docs.github.com/en/code-security/reference/code-scanning/sarif-files/sarif-support).

Projeyle gelen iş akışı Windows, Linux ve macOS'ta build/test/vet; ayrıca lint, biçim kontrolleri ve Linux race detector çalıştırır.

## Geliştirme

```sh
go build ./...
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
golangci-lint run
goimports -local github.com/toprakgureli/sizinti -w .
go test ./internal/scanner -run '^$' -bench . -benchmem
```

Race detector desteklenen bir C araç zinciri gerektirir. Testler sentetik yerel girdiler kullanır. GNU Make isteğe bağlıdır: `make build`, `make test`, `make lint`, `make run ARGS="."`, `make bench` ve `make fmt` aynı komutları toplar.

Araç sürümleri: goimports v0.36.0 (`golang.org/x/tools/cmd/goimports`), staticcheck v0.6.1 (`honnef.co/go/tools/cmd/staticcheck`) ve golangci-lint v2.12.2.

Kaydedilen paket kapsamı: entropy %100, allowlist %97,5, config %95,5, report %91,9, rules %89,3, scanner %88,9, CLI %88,5.

Ryzen 7 5800X3D üzerinde Windows/amd64 ve tek scanner worker ile alınmış yerel ölçüm:

```text
BenchmarkScan-16    222    5274423 ns/op    19.42 MB/s    239670 B/op    8240 allocs/op
```

Bu tek yerel ölçümdür, kıyas değildir. `-16` son eki scanner worker sayısını değil benchmark sürecinin ayarını gösterir.

Kod, [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)'ı izler. Tasarım kararları [DESIGN.md](DESIGN.md)'de açıklanır.

## İlham kaynakları

[Gitleaks](https://github.com/gitleaks/gitleaks) ve [TruffleHog](https://github.com/trufflesecurity/trufflehog) iş akışına ilham verdi. Daha geniş kapsam sunuyorlar; sizinti ise okunabilir Go ve açık Türk sağlayıcı bağlam kuralları etrafında geliştirdiğim odaklı uygulama.

## Yol haritası

- Git geçmişi taraması.
- Staged index taraması ve kurulabilir pre-commit hook.
- Daha fazla belgelenmiş Türk kuralı ve negatif örnek.
- Daha geniş kimlik bilgisi biçimleri ve daha kesin bağlam eşleştirmesi.

## Lisans

[MIT](LICENSE).
