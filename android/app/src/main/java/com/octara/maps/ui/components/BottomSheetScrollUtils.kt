package com.octara.maps.ui.components

import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.material3.BottomSheetDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.SheetState
import androidx.compose.material3.SheetValue
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.nestedscroll.NestedScrollConnection
import androidx.compose.ui.input.nestedscroll.NestedScrollSource
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.unit.Velocity
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InteractiveSheetDragHandle(
    sheetState: SheetState,
    modifier: Modifier = Modifier
) {
    val scope = rememberCoroutineScope()
    var dragDeltaAccumulator by remember { mutableFloatStateOf(0f) }

    Box(
        modifier = modifier
            .fillMaxWidth()
            .draggable(
                orientation = Orientation.Vertical,
                state = rememberDraggableState { delta ->
                    dragDeltaAccumulator += delta
                },
                onDragStarted = {
                    dragDeltaAccumulator = 0f
                },
                onDragStopped = { velocity ->
                    val delta = dragDeltaAccumulator
                    dragDeltaAccumulator = 0f
                    scope.launch {
                        try {
                            if (velocity < -300f || delta < -40f) {
                                sheetState.expand()
                            } else if (velocity > 300f || delta > 40f) {
                                sheetState.partialExpand()
                            }
                        } catch (_: Exception) {
                        }
                    }
                }
            ),
        contentAlignment = Alignment.Center
    ) {
        BottomSheetDefaults.DragHandle()
    }
}

@OptIn(ExperimentalMaterial3Api::class)
fun Modifier.bottomSheetContentScroll(
    scrollState: ScrollState,
    sheetState: SheetState? = null,
    onBoundaryDragUp: (() -> Unit)? = null,
    onBoundaryDragDown: (() -> Unit)? = null
): Modifier = composed {
    val currentSheetState by rememberUpdatedState(sheetState)
    val currentOnBoundaryDragUp by rememberUpdatedState(onBoundaryDragUp)
    val currentOnBoundaryDragDown by rememberUpdatedState(onBoundaryDragDown)
    val coroutineScope = rememberCoroutineScope()

    var startedAtTop by remember { mutableStateOf(true) }
    var startedAtBottom by remember { mutableStateOf(true) }
    var scrolledContentDownDuringGesture by remember { mutableStateOf(false) }
    var scrolledContentUpDuringGesture by remember { mutableStateOf(false) }
    var boundaryDragDownPx by remember { mutableFloatStateOf(0f) }
    var boundaryDragUpPx by remember { mutableFloatStateOf(0f) }

    val nestedScrollConnection = remember(scrollState) {
        object : NestedScrollConnection {
            override fun onPreScroll(available: Offset, source: NestedScrollSource): Offset {
                if (source != NestedScrollSource.UserInput) {
                    return Offset.Zero
                }

                if (available.y < 0f && scrollState.canScrollForward) {
                    scrolledContentUpDuringGesture = true
                    boundaryDragUpPx = 0f
                } else if (available.y > 0f && scrollState.canScrollBackward) {
                    scrolledContentDownDuringGesture = true
                    boundaryDragDownPx = 0f
                }

                return Offset.Zero
            }

            override fun onPostScroll(
                consumed: Offset,
                available: Offset,
                source: NestedScrollSource
            ): Offset {
                if (source != NestedScrollSource.UserInput) {
                    return available
                }

                if (consumed.y > 0f) {
                    scrolledContentDownDuringGesture = true
                    boundaryDragDownPx = 0f
                } else if (consumed.y < 0f) {
                    scrolledContentUpDuringGesture = true
                    boundaryDragUpPx = 0f
                }

                val sheet = currentSheetState

                if (available.y > 0f && !scrollState.canScrollBackward) {
                    boundaryDragDownPx += available.y

                    val thresholdPx = if (startedAtTop && !scrolledContentDownDuringGesture) 52f else 180f
                    if (boundaryDragDownPx >= thresholdPx) {
                        boundaryDragDownPx = 0f
                        scrolledContentDownDuringGesture = false
                        startedAtTop = true

                        if (currentOnBoundaryDragDown != null) {
                            currentOnBoundaryDragDown?.invoke()
                        } else if (sheet != null && sheet.currentValue == SheetValue.Expanded) {
                            coroutineScope.launch {
                                try {
                                    sheet.partialExpand()
                                } catch (_: Exception) {
                                }
                            }
                        }
                    }
                    return available
                }

                if (available.y < 0f && !scrollState.canScrollForward) {
                    boundaryDragUpPx += -available.y

                    val thresholdPx = if (startedAtBottom && !scrolledContentUpDuringGesture) 44f else 140f
                    if (boundaryDragUpPx >= thresholdPx) {
                        boundaryDragUpPx = 0f
                        scrolledContentUpDuringGesture = false
                        startedAtBottom = true

                        if (sheet != null && sheet.currentValue != SheetValue.Expanded) {
                            coroutineScope.launch {
                                try {
                                    sheet.expand()
                                } catch (_: Exception) {
                                }
                            }
                        } else {
                            currentOnBoundaryDragUp?.invoke()
                        }
                    }
                    return available
                }

                return Offset.Zero
            }

            override suspend fun onPreFling(available: Velocity): Velocity {
                return Velocity.Zero
            }

            override suspend fun onPostFling(consumed: Velocity, available: Velocity): Velocity {
                return available
            }
        }
    }

    this
        .pointerInput(scrollState) {
            awaitEachGesture {
                awaitFirstDown(requireUnconsumed = false, pass = PointerEventPass.Initial)
                startedAtTop = !scrollState.canScrollBackward
                startedAtBottom = !scrollState.canScrollForward
                scrolledContentDownDuringGesture = false
                scrolledContentUpDuringGesture = false
                boundaryDragDownPx = 0f
                boundaryDragUpPx = 0f
            }
        }
        .nestedScroll(nestedScrollConnection)
}
