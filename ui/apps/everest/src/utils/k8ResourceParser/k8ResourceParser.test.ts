import { cpuParser, isKubernetesUnit, memoryParser } from '.';

describe('cpu parser', () => {
  // pattern is [description, input, output]
  const tests: [string, string, number][] = [
    ['parses full numbers', '1', 1],
    ['parses floats (< 1)', '1.5', 1.5],
    ['parses floats (> 1)', '0.5', 0.5],
    ['parses strings with milli (m) unit (whole number)', '1000m', 1],
    ['parses strings with milli (m) unit (decimal number)', '1300m', 1.3],
    ['parses strings with milli (m) unit (< 1)', '300m', 0.3],
  ];

  tests.map((t) =>
    it(`${t[0]} (${t[1]} to ${t[2]})`, () => {
      expect(cpuParser(t[1])).toEqual(t[2]);
    })
  );
});

describe('isKubernetesUnit', () => {
  it('recognizes supported memory units', () => {
    expect(isKubernetesUnit('Gi')).toBe(true);
    expect(isKubernetesUnit('m')).toBe(true);
  });

  it('rejects unsupported badge values', () => {
    expect(isKubernetesUnit('GB')).toBe(false);
    expect(isKubernetesUnit('cores')).toBe(false);
  });
});

describe('memory parser', () => {
  it('correctly parses memory strings', () => {
    expect(memoryParser('1')).toEqual({ value: 1, originalUnit: '' });
    expect(memoryParser('1k', 'G')).toEqual({
      value: 1 * 10 ** -6,
      originalUnit: 'k',
    });
    expect(memoryParser('1G', 'Gi')).toEqual({
      value: 10 ** 9 / 1024 ** 3,
      originalUnit: 'G',
    });
    expect(memoryParser('1G', 'G')).toEqual({ value: 1, originalUnit: 'G' });
    expect(memoryParser('3000m', 'Gi')).toEqual({
      value: 3 / 1024 ** 3,
      originalUnit: 'm',
    });
    expect(memoryParser('1073741824000m', 'Gi')).toEqual({
      value: 1,
      originalUnit: 'm',
    });
  });
});
