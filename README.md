<p align="center">
<img style="vertical-align:right" width="128" height="128"  alt="" src="https://s469vla.storage.yandex.net/rdisk/437d69e71b0b8d00269f3c2fe8098e332caad68c1ac6aea245a68a43537756be/66b40312/eZLdwrKxPcuKfu4_b1Tf1cy1pzAql_jud-4O0NRYyhrvFL1qVm38d9mMOh9HL5ZdMxUXLZaiDSxdu-iLmt317w==?uid=1523954673&filename=test.min.png&disposition=inline&hash=&limit=0&content_type=image%2Fpng&owner_uid=1523954673&fsize=102858&hid=16b24dca1f1e72448503ac0e5eb77574&media_type=image&tknv=v2&etag=2fe269af80b352af4b2bea8cb9f022af&ts=61f2043d96880&s=2226420a59ce45ae31e37f23a690dfdb92ebeec58b82e120c24db36efbd0003a&pb=U2FsdGVkX18GEI_nY2MxGE6Z219twwXDT-l17-PAmr2HiwnvqCe5ylXXs8-uni7L0qw_ii7wcHtd7VXnHbHymufFCqPj3KmbDJO9JE6DRYk">
</p>

<h1 align="center">T-Short</h1>

### Маленькая либа для написания удобных unit тестов

Основная идея в том чтобы уменьшить количество кода и копипаста, при этом сохранив читабельность и интуитивность описания тест кейсов

Пример в [example](./example)

---

### Функция Copy которая создает детальную копию входного значения

---

### Генератор моков

    Входные параметры
    --outdir - Папка куда будут генерироваться  файлы, если пусто то создает папку mocks в дериктории файла
    --outfilename - Имя выходного файла, если пустое то mock+имя файла
    --outpkg - Имя выходного пакета, если пусто то mock+имя пакета
    --intgen - Через запятую перечисление имен интерфейсов, если пусто то генерирует все интерфейсы
