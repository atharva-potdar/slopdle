/** Backdrop click-to-close for modals. Fires only when the backdrop itself is
 * clicked, so inner content needs no stopPropagation handler. Keyboard users
 * close via the global Escape handler each view already provides, plus the
 * dialog's own buttons — hence no on:click on static divs (a11y-clean). */
export function backdrop(node: HTMLElement, onDismiss: () => void) {
  const click = (e: MouseEvent) => {
    if (e.target === node) onDismiss();
  };
  node.addEventListener('click', click);
  return {
    destroy() {
      node.removeEventListener('click', click);
    },
  };
}
