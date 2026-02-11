/*
Converts a flat object with dot-notation keys to a nested object.
This is needed for correct reading of Zod and CEL validation errors.
Example: { "spec.replica.nodes": 3 } => { spec: { replica: { nodes: 3 } } }
*/

export const convertToNestedObject = (
  flatObj: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, any> = {};

  Object.entries(flatObj).forEach(([path, value]) => {
    const keys = path.split('.');
    let current = result;

    keys.forEach((key, index) => {
      if (index === keys.length - 1) {
        // Last key - set the value
        current[key] = value;
      } else {
        // Intermediate key - ensure object exists
        if (!current[key] || typeof current[key] !== 'object') {
          current[key] = {};
        }
        current = current[key];
      }
    });
  });

  return result;
};
