package com.octara.maps.ui.screens.map

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SheetValue
import androidx.compose.material3.Surface
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.material3.rememberStandardBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.ui.components.AppSearchBar
import com.octara.maps.ui.theme.AppDimensions
import org.maplibre.android.geometry.LatLng

@Composable
fun TransitMapSheetContent(
    onSearchClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        AppSearchBar(
            active = false,
            onClick = onSearchClick
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransitMapScreen(
    modifier: Modifier = Modifier
) {
    val router = LocalRouter.current
    val configuration = LocalConfiguration.current
    val density = LocalDensity.current

    val currentScreen = router.activeMapScreen
    val isDismissibleState = rememberUpdatedState(currentScreen.isDismissible)

    var targetCoords by remember { mutableStateOf<LatLng?>(null) }
    var targetTitle by remember { mutableStateOf<String?>(null) }

    val sheetPeekHeight = remember(currentScreen, configuration.screenHeightDp) {
        currentScreen.sheetPeekHeight(configuration.screenHeightDp.dp)
    }
    val sheetPeekHeightPx = with(density) { sheetPeekHeight.roundToPx() }

    val sheetState = rememberStandardBottomSheetState(
        initialValue = SheetValue.PartiallyExpanded,
        confirmValueChange = { newValue ->
            if (newValue == SheetValue.Hidden) {
                isDismissibleState.value
            } else {
                true
            }
        },
        skipHiddenState = false
    )
    val scaffoldState = rememberBottomSheetScaffoldState(
        bottomSheetState = sheetState
    )

    val handleClose: () -> Unit = {
        targetCoords = null
        targetTitle = null
        router.goBack()
        Unit
    }

    LaunchedEffect(currentScreen) {
        targetCoords = currentScreen.mapTarget
        targetTitle = currentScreen.mapTitle
        try {
            sheetState.partialExpand()
        } catch (_: Exception) {}
    }

    BottomSheetScaffold(
        scaffoldState = scaffoldState,
        sheetShape = AppDimensions.BottomSheetShape,
        sheetPeekHeight = sheetPeekHeight,
        sheetShadowElevation = AppDimensions.BottomSheetElevation,
        sheetContent = {
            key(currentScreen) {
                currentScreen.SheetContent(
                    sheetState = sheetState,
                    onClose = handleClose,
                    onUpdateTarget = { coords, title ->
                        if (targetCoords == null) {
                            targetCoords = coords
                            targetTitle = title
                        }
                    }
                )
            }
        },
        modifier = modifier.fillMaxSize()
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            TransitMap(
                modifier = Modifier.fillMaxSize(),
                cameraTarget = targetCoords,
                cameraZoom = currentScreen.cameraZoom,
                bottomPaddingPx = sheetPeekHeightPx,
                markerPosition = targetCoords,
                markerTitle = targetTitle
            )

            AnimatedVisibility(
                visible = currentScreen.showBackButton,
                enter = fadeIn(),
                exit = fadeOut(),
                modifier = Modifier
                    .statusBarsPadding()
                    .padding(start = 16.dp, top = 8.dp)
                    .align(Alignment.TopStart)
            ) {
                Surface(
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.surface.copy(alpha = 0.92f),
                    shadowElevation = 4.dp
                ) {
                    IconButton(
                        onClick = handleClose,
                        modifier = Modifier.size(42.dp)
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Retour",
                            tint = MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }
        }
    }
}
