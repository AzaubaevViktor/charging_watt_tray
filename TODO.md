# TODO

## Done (Go port)
- [x] Переписано на Go (`fogleman/gg` + `getlantern/systray`), по образцу
      соседнего `claude_limits`. SMC и IOReport — через cgo (IOKit / dlopen
      `libIOReport.dylib`); `ioreg`/`top` — через subprocess. На реальном
      железе цифры сходятся (PSTR=System=CPU+GPU+Other).
- [x] Разбито на модули: `smc.go`, `ioreport.go`, `battery.go`, `top.go`,
      `format.go`, `icon.go`, `app.go`, `menu.go`, `main.go`. Чистая логика
      (`parseTop`, `fmtTime*`, `wattTitle`) покрыта тестами.
- [x] Цифра ватт фиксированной ширины в баре: целая часть дополняется
      figure-space (U+2007, ширина цифры) до 2 знаков — `wattTitle` в
      `format.go`. "5W" и "12W" занимают одну ширину, бар не дёргается.
- [x] Код упрощён: убран ленивый ioreg-`PowerTelemetryData` fallback (на
      Apple Silicon SMC всегда есть), вся C-возня спрятана в cgo-обёртки.

## Open (после порта)
- [ ] Правое выравнивание ватт в "Top usage": в Python был NSAttributedString
      с табстопом; systray рисует только plain text, так что колонка теперь
      просто после имени процесса. Вернуть выравнивание = свой NSView/attrs.

## Done
- [x] Real-time мощность без лага и без sudo — чтение SMC (см. `smc.py`):
      PSTR=система, PDTR=адаптер, PBAT=батарея. Заменило медленный
      PowerTelemetryData из ioreg (заодно ушло "большое число" = unsigned
      обёртка отрицательного SystemLoad на батарее).
- [x] Заряд % и время до полного/пустого (CurrentCapacity, AvgTimeToFull /
      AvgTimeToEmpty из ioreg) — показываются в меню.
- [x] Среднее время обновления данных (по последним 10 изменениям) — в меню.
- [x] Иконка батареи горизонтальная, генерится на лету; шрифт бара уменьшен.

## Done (cont.)
- [x] Top energy users: значения выровнены справа (NSAttributedString +
      правый табстоп). Per-process ватты нормируются на реальную мощность
      CPU+GPU из IOReport (`ioreport.py`, группа "Energy Model", без sudo),
      а дисплей/периферия/DRAM/база показаны отдельной строкой "Other"
      (= PSTR − CPU+GPU), чтобы не вешать их на процессы.

## Open
- [ ] (опц.) Точные per-process ватты через powermetrics (sudo) — даст
      реальное CPU/GPU/ANE-разбиение вместо пропорции по energy-impact.
      Сейчас показываем "~" (приближение, нормировка на CPU+GPU). Нужно
      только если важна точность распределения между процессами.

- [ ] Показывать абсолютное "обновлено N сек назад" (сейчас есть только
      среднее). С SMC данные real-time (~1с), так что это скорее индикатор
      живости; можно убрать, если не нужно.

- [ ] Уровень заряда в иконке и при работе от адаптера (сейчас при зарядке
      иконка — молния, без уровня). Можно совместить: батарея с молнией.
