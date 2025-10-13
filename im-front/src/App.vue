<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script setup>
// 主应用组件
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  overflow-y: auto;
  scroll-behavior: smooth; /* 全局平滑滚动 */
  overscroll-behavior: contain; /* 防止滚动链到嵌套容器导致抖动 */
  /* 放松分页吸附以提升滚动顺滑度 */
  scroll-snap-type: y proximity;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background-color: #f5f5f5;
}

#app {
  min-height: 100vh;
  /* 移除内部滚动，避免与窗口滚动叠加导致卡顿 */
  overflow: visible;
}

button {
  transition: all 0.2s ease;
}

button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

input, textarea {
  font-family: inherit;
}

input:focus, textarea:focus {
  outline: none;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
