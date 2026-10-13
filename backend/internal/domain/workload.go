package domain

// Política de carga horaria (validaciones bloqueantes, RN-14 a RN-19, RF-05).
//
// Se evalúa sobre el conjunto de vinculaciones activas de la persona más la
// vinculación candidata:
//   - asistencia graduada: máximo 22 h/semana;
//   - monitorías: máximo 3 simultáneas y 12 h/semana;
//   - regla mixta: horas de monitoría <= ceil(40 % de las horas de AG).
//
// Ejemplos de aceptación: AG 22 h -> máximo 9 h de monitoría;
// AG 12 h -> máximo 5 h de monitoría.
