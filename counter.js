document.addEventListener("DOMContentLoaded", () => {
    const MAX_WATER_LEVEL = 0;
    const MIN_WATER_LEVEL = 100;
    const DEFAULT_WATER_LEVEL = 60;

    const button = document.getElementById("waterDropBtn");
    const waterLevel = document.getElementById("waterLevel");
    waterLevel.setAttribute("y", DEFAULT_WATER_LEVEL);

    let animationFrame;
    let holdTimeout;
    let startTime;
    let isHolding = false;
    let isAnimating = false;
    let decrementFinished = false;

    function animateTo(yValue, duration, callback) {
        const startY = parseFloat(waterLevel.getAttribute("y"));
        const startTime = performance.now();

        function step(timestamp) {
            const elapsed = timestamp - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const newY = startY + (yValue - startY) * progress;

            waterLevel.setAttribute("y", newY);

            if (progress < 1) {
                animationFrame = requestAnimationFrame(step);
            } else {
                if (callback) {
                    callback();
                }
            }
        }

        cancelAnimationFrame(animationFrame);
        animationFrame = requestAnimationFrame(step);
    }

    button.addEventListener("mousedown", () => {
        if (isAnimating) {
            // Reset animation if clicked again
            cancelAnimationFrame(animationFrame);
            animateTo(DEFAULT_WATER_LEVEL, 800, () => {
                isAnimating = false;
            });
            return;
        }

        startTime = performance.now();
        isHolding = true;

        function increaseWaterLevel() {
            const elapsed = performance.now() - startTime;
            let newY = DEFAULT_WATER_LEVEL + (elapsed / 2000) * DEFAULT_WATER_LEVEL; // Takes 2 seconds to reach y=100

            if (newY >= MIN_WATER_LEVEL) {
                waterLevel.setAttribute("y", MIN_WATER_LEVEL);
                // isHolding = false;
                console.log("DECREMENT");
                decrementFinished = true;
                animateTo(DEFAULT_WATER_LEVEL, 0);
                // htmx.ajax("POST", "/decrement", { target: "#result", swap: "innerHTML" });
            } else {
                waterLevel.setAttribute("y", newY);
                holdTimeout = requestAnimationFrame(increaseWaterLevel);
            }
        }

        holdTimeout = requestAnimationFrame(increaseWaterLevel);
    });

    button.addEventListener("mouseup", () => {
        // detect click
        if (isHolding && startTime && performance.now() - startTime < 200) {
            cancelAnimationFrame(holdTimeout);
            isHolding = false;
        }

        if (isHolding) {
            // If released before reaching 100, reset
            cancelAnimationFrame(holdTimeout);
            animateTo(DEFAULT_WATER_LEVEL, 300);
            isHolding = false;
        } else if (!isAnimating) {
            // If it wasn't a hold, do the normal animation
            isAnimating = true;
            animateTo(MAX_WATER_LEVEL, 500, () => {
                isAnimating = false;
                console.log("INCREMENT");
                // htmx.ajax("POST", "/increment", { target: "#result", swap: "innerHTML" });
                animateTo(DEFAULT_WATER_LEVEL, 0);
            });
        }

        isHolding = false;
    });

    button.addEventListener("mouseleave", () => {
        if (isHolding && !decrementFinished) {
            cancelAnimationFrame(holdTimeout);
            animateTo(DEFAULT_WATER_LEVEL, 300);
            isHolding = false;
        }
    });
});