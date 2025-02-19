function getSpeed(distance, time) {
    return distance / time;
}

document.addEventListener("DOMContentLoaded", () => {
    /*** CONSTANTS ***/
    const MAX_WATER_LEVEL = 3;
    const MIN_WATER_LEVEL = 92;
    const DEFAULT_WATER_LEVEL = 65;
    const FAST_SPEED = getSpeed(60, 1000);
    const SLOW_SPEED = getSpeed(40, 2000);

    /*** ELEMENTS ***/
    const button = document.getElementById("waterDropBtn");
    const waterLevelElem = document.getElementById("waterLevel");

    const REQUEST_OPTIONS = {
        hxTarget: button.getAttribute("hx-target"),
        hxSwap: button.getAttribute("hx-swap"),
        amount: button.getAttribute("data-amount"),
    }

    const makeRequest = typeof htmx !== 'undefined' && htmx && REQUEST_OPTIONS.amount;

    /*** STATE ***/
    let waterLevel = DEFAULT_WATER_LEVEL;
    let animationFrame = null;
    let isHolding = false;
    let isIncrementing = false;
    let startTime = null;

    waterLevelElem.setAttribute("y", DEFAULT_WATER_LEVEL);

    /*** FUNCTIONS ***/
    const setWaterLevel = (level) => {
        waterLevel = level;
        waterLevelElem.setAttribute("y", level);
    };

    function animateTo(target, speed, callback) {
        cancelAnimationFrame(animationFrame);

        // Instantly set water level if speed is 0 or already at target
        if (speed === 0 || target === waterLevel) {
            setWaterLevel(target);
            callback?.();
            return;
        }

        const start = waterLevel;
        const animStartTime = performance.now();
        const direction = target > start ? 1 : -1; // Moving up or down
        const absSpeed = Math.abs(speed); // Ensure speed is always positive

        function step(timestamp) {
            const elapsed = timestamp - animStartTime;
            const newY = start + direction * absSpeed * elapsed;

            if ((direction === 1 && newY >= target) || (direction === -1 && newY <= target)) {
                setWaterLevel(target);
                callback?.();
                return;
            }

            setWaterLevel(newY);
            animationFrame = requestAnimationFrame(step);
        }

        animationFrame = requestAnimationFrame(step);
    }

    /*** EVENT HANDLERS ***/
    function handleStart() {
        if (isIncrementing) {
            // If already incrementing, reset to default
            animateTo(DEFAULT_WATER_LEVEL, SLOW_SPEED, () => (isIncrementing = false));
            return;
        }

        startTime = performance.now();
        isHolding = true;

        // Start decreasing water level
        animateTo(MIN_WATER_LEVEL, SLOW_SPEED, () => {
            console.log("DECREMENT");
            if (makeRequest) {
                htmx.ajax("POST", "/decrement", {
                    target: REQUEST_OPTIONS.hxTarget,
                    swap: REQUEST_OPTIONS.hxSwap,
                    vals: {amount: REQUEST_OPTIONS.amount}
                });
            }
            animateTo(DEFAULT_WATER_LEVEL, 0);
        });
    }

    const handleStop = (isLeave) => () => {
        if (!isHolding) return;

        // detect click
        const isClick = !isLeave && performance.now() - startTime < 200;
        isHolding = false;

        if (isClick) {
            isIncrementing = true;
            animateTo(MAX_WATER_LEVEL, FAST_SPEED, () => {
                console.log("INCREMENT");
                if (makeRequest) {
                    htmx.ajax("POST", "/increment", {
                        target: REQUEST_OPTIONS.hxTarget,
                        swap: REQUEST_OPTIONS.hxSwap,
                        vals: {amount: REQUEST_OPTIONS.amount}
                    });
                }
                isIncrementing = false;
                animateTo(DEFAULT_WATER_LEVEL, 0);
            });
        } else {
            animateTo(DEFAULT_WATER_LEVEL, FAST_SPEED);
        }
    }

    /*** EVENT LISTENERS ***/
    button.addEventListener("mousedown", handleStart);
    button.addEventListener("mouseup", handleStop(false));
    button.addEventListener("mouseleave", handleStop(true));

    // Touch Events for Mobile
    button.addEventListener("touchstart", (event) => {
        event.preventDefault(); // Prevents ghost clicks
        handleStart();
    });
    button.addEventListener("touchend", handleStop(false));
    button.addEventListener("touchcancel", handleStop(true));
});
