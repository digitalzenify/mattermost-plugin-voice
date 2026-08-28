import {useEffect, useState, type RefObject} from 'react';

export type PopoverPosition = {
    top: number;
    left: number;
    width: number;
};

const MARGIN = 8;

/**
 * Tracks where a fixed-position popover should sit, right-aligned above the given anchor
 * element, clamped to stay fully inside the viewport. Recomputed on resize/scroll and on
 * viewport size changes (e.g. a mobile on-screen keyboard opening), since the popover is
 * rendered through a portal and can't rely on CSS positioning relative to its DOM parent.
 */
export function useAnchoredPopoverPosition(
    anchorRef: RefObject<HTMLElement>,
    active: boolean,
    preferredWidth = 300,
): PopoverPosition | null {
    const [position, setPosition] = useState<PopoverPosition | null>(null);

    useEffect(() => {
        if (!active) {
            setPosition(null);
            return undefined;
        }

        const update = () => {
            const anchor = anchorRef.current;
            if (!anchor) {
                return;
            }
            const rect = anchor.getBoundingClientRect();
            const viewportWidth = window.visualViewport?.width ?? window.innerWidth;

            const width = Math.min(preferredWidth, viewportWidth - (MARGIN * 2));
            const left = Math.min(
                Math.max(MARGIN, rect.right - width),
                viewportWidth - width - MARGIN,
            );
            const top = Math.max(MARGIN, rect.top - MARGIN);

            setPosition({top, left, width});
        };

        update();

        window.addEventListener('resize', update);
        window.addEventListener('scroll', update, true);
        window.visualViewport?.addEventListener('resize', update);
        window.visualViewport?.addEventListener('scroll', update);

        return () => {
            window.removeEventListener('resize', update);
            window.removeEventListener('scroll', update, true);
            window.visualViewport?.removeEventListener('resize', update);
            window.visualViewport?.removeEventListener('scroll', update);
        };
    }, [active, anchorRef, preferredWidth]);

    return position;
}
